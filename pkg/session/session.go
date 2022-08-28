package session

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/yubo/apiserver/pkg/models"
	"github.com/yubo/apiserver/pkg/session/types"
	"github.com/yubo/apiserver/pkg/storage"
	"github.com/yubo/golib/api"
	"github.com/yubo/golib/api/errors"
	"github.com/yubo/golib/orm"
	"github.com/yubo/golib/util"
	"github.com/yubo/golib/util/clock"
	"k8s.io/klog/v2"
)

const (
	DefCookieName = "sid"
)

// config {{{
func NewConfig() *Config {
	return &Config{
		CookieName:     DefCookieName,
		SidLength:      32,
		HttpOnly:       true,
		GcInterval:     api.NewDuration("600s"),
		CookieLifetime: api.NewDuration("16h"),
		MaxIdleTime:    api.NewDuration("1h"),
		TableName:      "session",
	}
}

type Config struct {
	CookieName     string       `json:"cookieName"`
	SidLength      int          `json:"sidLength"`
	HttpOnly       bool         `json:"httpOnly"`
	Domain         string       `json:"domain"`
	GcInterval     api.Duration `json:"gcInterval"`
	CookieLifetime api.Duration `json:"cookieLifetime"`
	MaxIdleTime    api.Duration `json:"maxIdleTime"`
	TableName      string       `json:"tableName"`
}

func (p *Config) Validate() error {
	if p == nil {
		return nil
	}

	if p.SidLength <= 0 {
		return fmt.Errorf("invalid sid length %d", p.SidLength)
	}

	if p.CookieName == "" {
		p.CookieName = DefCookieName
	}

	return nil
}

// }}}

//  sessionManager {{{
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
		session = NewSessionConn()
	}

	return &sessionManager{
		session: session,
		config:  cf,
		Options: opts,
	}, nil
}

type sessionManager struct {
	Options
	session *SessionConn
	config  *Config
	once    sync.Once
}

func (p *sessionManager) GC() {
	p.once.Do(func() {
		cookieName := p.config.CookieName
		gcInterval := p.config.GcInterval.Duration
		maxIdleTime := p.config.MaxIdleTime.Duration

		util.UntilWithTick(
			func() {
				if err := p.session.Clean(p.ctx, p.clock.Now().Add(-maxIdleTime).Unix(), cookieName); err != nil {
					klog.Warningf("session.Clean() err %s", err)
				}
			},
			p.clock.NewTicker(gcInterval).C(),
			p.ctx.Done(),
		)

	})
}

// SessionStart generate or read the session id from http request.
// if session id exists, return SessionStore with this id.
func (p *sessionManager) Start(w http.ResponseWriter, req *http.Request) (sess types.SessionContext, err error) {
	var sid string

	if sid, err = p.getSid(req); err != nil {
		return
	}

	if sid != "" {
		if sess, err := p.getSessionStore(req.Context(), sid, false); err == nil {
			return sess, nil
		}
	}

	// Generate a new session
	sid = util.RandString(p.config.SidLength)

	sess, err = p.getSessionStore(req.Context(), sid, true)
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
	req.AddCookie(cookie)
	return
}

func (p *sessionManager) Stop() {
	if p.cancel != nil {
		p.cancel()
	}
}

func (p *sessionManager) Destroy(w http.ResponseWriter, req *http.Request) error {
	cookie, err := req.Cookie(p.config.CookieName)
	if err != nil || cookie.Value == "" {
		return errors.NewUnauthorized("Have not login yet")
	}

	sid, _ := url.QueryUnescape(cookie.Value)
	if err := p.session.Delete(req.Context(), sid); err != nil {
		return err
	}

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
	sc, err := p.session.Get(ctx, sid)
	if errors.IsNotFound(err) && create {
		ts := p.clock.Now().Unix()
		sc = &types.SessionConn{
			Sid:        sid,
			CookieName: p.config.CookieName,
			CreatedAt:  ts,
			UpdatedAt:  ts,
			Data:       make(map[string]string),
		}
		err = p.session.Create(ctx, sc)
	}
	if err != nil {
		return nil, err
	}
	return &sessionContext{manager: p, conn: sc}, nil
}

// }}}

// sessionContext {{{

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
	return p.manager.session.Update(context.Background(), p.conn)
}

// }}}

// Options {{{

type Options struct {
	ctx      context.Context
	cancel   context.CancelFunc
	clock    clock.Clock
	sessions *SessionConn
}

type Option func(*Options)

func WithCtx(ctx context.Context) Option {
	return func(o *Options) {
		o.ctx = ctx
		o.cancel = nil
	}
}

func WithModel(m *SessionConn) Option {
	return func(o *Options) {
		o.sessions = m
	}
}

func WithClock(clock clock.Clock) Option {
	return func(o *Options) {
		o.clock = clock
	}
}

// }}}

// sessionConn {{{

// pkg/registry/rbac/role/storage/storage.go
// pkg/registry/rbac/rest/storage_rbac.go
func NewSessionConn() *SessionConn {
	o := &SessionConn{DB: models.DB()}
	return o
}

type SessionConn struct {
	orm.DB
}

func (p *SessionConn) Name() string {
	return "session_conn"
}

func (p *SessionConn) NewObj() interface{} {
	return &types.SessionConn{}
}

func (p *SessionConn) Create(ctx context.Context, obj *types.SessionConn) error {
	return p.Insert(ctx, obj, orm.WithTable(p.Name()))
}

// Get retrieves the session from the db for a given sid.
func (p *SessionConn) Get(ctx context.Context, sid string) (ret *types.SessionConn, err error) {
	err = p.Query(ctx, "select * from session_conn where sid=?", sid).Row(&ret)
	return
}

// List lists all sessions in the indexer.
func (p *SessionConn) List(ctx context.Context, o *storage.ListOptions, opts ...orm.Option) (list []types.SessionConn, err error) {
	err = p.DB.List(ctx, &list, append(opts,
		orm.WithTable(p.Name()),
		orm.WithTotal(o.Total),
		orm.WithSelector(o.Query),
		orm.WithOrderby(o.Orderby...),
		orm.WithLimit(o.Offset, o.Limit))...,
	)
	return
}

func (p *SessionConn) Update(ctx context.Context, obj *types.SessionConn) error {
	return p.DB.Update(ctx, obj, orm.WithTable(p.Name()))
}

func (p *SessionConn) Delete(ctx context.Context, sid string) error {
	_, err := p.Exec(ctx, "delete from session_conn where sid=?", sid)
	return err
}

func (p *SessionConn) Clean(ctx context.Context, expiresAt int64, cookieName string) error {
	_, err := p.Exec(ctx, "delete from session_conn where updated_at<? and cookie_name=?", expiresAt, cookieName)
	return err
}

// }}}
