package session

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/yubo/apiserver/pkg/session/types"
	"github.com/yubo/apiserver/pkg/storage"
	"github.com/yubo/golib/api/errors"
	"github.com/yubo/golib/util"
	"github.com/yubo/golib/util/clock"
	"k8s.io/klog/v2"
)

func NewSessionManager(cf *Config, optsInput ...Option) (types.SessionManager, error) {
	opts := Options{}

	for _, opt := range optsInput {
		opt(&opts)
	}

	if opts.ctx == nil {
		opts.ctx = context.Background()
	}
	opts.ctx, opts.cancel = context.WithCancel(opts.ctx)

	if opts.clock == nil {
		opts.clock = clock.RealClock{}
	}

	session := opts.sessions
	if session == nil {
		session = NewSession()
	}

	return &sessionManager{
		Session: session,
		config:  cf,
		Options: opts,
	}, nil
}

//type sessionConn struct {
//	Sid        string `sql:"name,where,primary_key"`
//	UserName   string
//	Data       map[string]string
//	CookieName string
//	CreatedAt  int64
//	UpdatedAt  int64
//}

type sessionManager struct {
	types.Session
	Options
	config *Config
	once   sync.Once
}

func (p *sessionManager) GC() {
	p.once.Do(func() {
		cf := p.config
		opts := p.Options
		fn := func() {
			query := fmt.Sprintf("updated_at<%d,cookie_name=%s",
				opts.clock.Now().Add(-cf.MaxIdleTime.Duration).Unix(),
				cf.CookieName,
			)
			list, err := p.List(p.ctx, storage.ListOptions{Query: query})
			if err != nil {
				klog.Warningf("list err %s", err)
				return
			}

			klog.V(3).InfoS("list", "query", query, "list", len(list))

			for _, v := range list {
				p.Delete(p.ctx, v.Sid)
			}
		}

		util.UntilWithTick(fn,
			opts.clock.NewTicker(cf.GcInterval.Duration).C(),
			opts.ctx.Done())

	})
}

// SessionStart generate or read the session id from http request.
// if session id exists, return SessionStore with this id.
func (p *sessionManager) Start(w http.ResponseWriter, r *http.Request) (sess types.SessionContext, err error) {
	var sid string

	if sid, err = p.getSid(r); err != nil {
		return
	}

	ctx := r.Context()

	if sid != "" {
		if sess, err := p.getSessionStore(ctx, sid, false); err == nil {
			return sess, nil
		}
	}

	// Generate a new session
	sid = util.RandString(p.config.SidLength)

	sess, err = p.getSessionStore(ctx, sid, true)
	if err != nil {
		return nil, err
	}
	cookie := &http.Cookie{
		Name:     p.config.CookieName,
		Value:    url.QueryEscape(sid),
		Path:     "/",
		HttpOnly: p.config.HttpOnly,
		Domain:   p.config.Domain,
	}
	if n := int(p.config.CookieLifetime.Duration.Seconds()); n > 0 {
		cookie.MaxAge = n
	}
	http.SetCookie(w, cookie)
	r.AddCookie(cookie)
	return
}

func (p *sessionManager) Stop() {
	if p.cancel != nil {
		p.cancel()
	}
}

func (p *sessionManager) Destroy(w http.ResponseWriter, r *http.Request) error {
	cookie, err := r.Cookie(p.config.CookieName)
	if err != nil || cookie.Value == "" {
		return errors.NewUnauthorized("Have not login yet")
	}

	sid, _ := url.QueryUnescape(cookie.Value)
	p.Delete(r.Context(), sid)

	cookie = &http.Cookie{Name: p.config.CookieName,
		Path:     "/",
		HttpOnly: p.config.HttpOnly,
		Expires:  p.clock.Now(),
		MaxAge:   -1}

	http.SetCookie(w, cookie)
	return nil
}

func (p *sessionManager) Get(ctx context.Context, sid string) (types.SessionContext, error) {
	return p.getSessionStore(ctx, sid, true)
}

func (p *sessionManager) Exist(sid string) bool {
	_, err := p.Get(context.Background(), sid)
	return !errors.IsNotFound(err)
}

func (p *sessionManager) getSid(r *http.Request) (string, error) {
	cookie, err := r.Cookie(p.config.CookieName)
	if err != nil || cookie.Value == "" {
		return "", nil
	}

	return url.QueryUnescape(cookie.Value)
}

func (p *sessionManager) getSessionStore(ctx context.Context, sid string, create bool) (types.SessionContext, error) {
	sc, err := p.Session.Get(ctx, sid)
	if errors.IsNotFound(err) && create {
		ts := p.clock.Now().Unix()
		sc = &types.SessionConn{
			Sid:        sid,
			CookieName: p.config.CookieName,
			CreatedAt:  ts,
			UpdatedAt:  ts,
			Data:       make(map[string]string),
		}
		_, err = p.Create(ctx, sc)
	}
	if err != nil {
		return nil, err
	}
	return &sessionContext{manager: p, conn: sc}, nil
}

// sessionContext mysql sessionContext store
type sessionContext struct {
	sync.RWMutex
	conn    *types.SessionConn
	manager *sessionManager
}

// Set value in mysql session.
// it is temp value in map.
func (p *sessionContext) Set(key, value string) error {
	p.Lock()
	defer p.Unlock()

	switch strings.ToLower(key) {
	case "username":
		p.conn.UserName = value
	default:
		p.conn.Data[key] = value
	}
	return nil
}

// Get value from mysql session
func (p *sessionContext) Get(key string) string {
	p.RLock()
	defer p.RUnlock()

	switch strings.ToLower(key) {
	case "username":
		return p.conn.UserName
	default:
		return p.conn.Data[key]
	}
}

func (p *sessionContext) CreatedAt() time.Time {
	return time.Unix(p.conn.CreatedAt, 0)
}

// Delete value in mysql session
func (p *sessionContext) Delete(key string) error {
	p.Lock()
	defer p.Unlock()

	delete(p.conn.Data, key)
	return nil
}

// Reset clear all values in mysql session
func (p *sessionContext) Reset() error {
	p.Lock()
	defer p.Unlock()

	p.conn.UserName = ""
	p.conn.Data = make(map[string]string)
	return nil
}

// Sid get session id of this mysql session store
func (p *sessionContext) Sid() string {
	return p.conn.Sid
}

func (p *sessionContext) Update(w http.ResponseWriter) error {
	p.conn.UpdatedAt = p.manager.clock.Now().Unix()
	_, err := p.manager.Session.Update(context.Background(), p.conn)
	return err
}
