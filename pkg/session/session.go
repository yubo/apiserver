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
	"github.com/yubo/golib/api"
	"github.com/yubo/golib/api/errors"
	"github.com/yubo/golib/util"
	"github.com/yubo/golib/util/clock"
	"k8s.io/klog/v2"
)

const (
	CookieName = "sid"
)

func newConfig() *Config {
	return &Config{
		CookieName:     CookieName,
		SidLength:      32,
		HttpOnly:       true,
		GcInterval:     api.NewDuration("600s"),
		CookieLifetime: api.NewDuration("16h"),
		MaxIdleTime:    api.NewDuration("1h"),
		TableName:      "session",
	}
}

// config {{{
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
		p.CookieName = CookieName
	}

	return nil
}

// }}}

func newManager(ctx context.Context, cf *Config, c clock.Clock) *manager {
	if util.IsNil(c) {
		c = clock.RealClock{}
	}

	return &manager{
		ctx:     ctx,
		config:  cf,
		session: NewSessionConn(),
		clock:   c,
	}
}

// manager {{{
type manager struct {
	ctx     context.Context
	session *SessionConn
	config  *Config
	once    sync.Once
	clock   clock.Clock
}

func (p *manager) GC() {
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
func (p *manager) Start(w http.ResponseWriter, req *http.Request) (sess types.Session, err error) {
	var sid string

	if sid, err = p.getSid(req); err != nil {
		return
	}

	if sid != "" {
		if sess, err := p.getConnection(req.Context(), sid, false); err == nil {
			return sess, nil
		}
	}

	// Generate a new session
	sid = util.RandString(p.config.SidLength)

	sess, err = p.getConnection(req.Context(), sid, true)
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

func (p *manager) Destroy(w http.ResponseWriter, req *http.Request) error {
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

func (p *manager) Get(ctx context.Context, sid string) (types.Session, error) {
	return p.getConnection(ctx, sid, true)
}

func (p *manager) Exist(sid string) bool {
	_, err := p.Get(context.Background(), sid)
	return !errors.IsNotFound(err)
}

func (p *manager) getSid(r *http.Request) (string, error) {
	cookie, err := r.Cookie(p.config.CookieName)
	if err != nil || cookie.Value == "" {
		return "", nil
	}

	return url.QueryUnescape(cookie.Value)
}

func (p *manager) getConnection(ctx context.Context, sid string, create bool) (types.Session, error) {
	return getConnection(ctx, p.session, p.config.CookieName, sid, create, p.clock)
}

//}}}

func getConnection(ctx context.Context, session *SessionConn, cookieName, sid string, create bool, clock clock.Clock) (*connection, error) {
	sc, err := session.Get(ctx, sid)
	if errors.IsNotFound(err) && create {
		ts := clock.Now().Unix()
		sc = &types.SessionConn{
			Sid:        sid,
			CookieName: cookieName,
			CreatedAt:  ts,
			UpdatedAt:  ts,
			Data:       make(url.Values),
		}
		err = session.Create(ctx, sc)
	}
	if err != nil {
		return nil, err
	}

	return &connection{session: session, conn: sc, clock: clock}, nil

}

// connection {{{

// connection mysql connection store
type connection struct {
	conn    *types.SessionConn
	session *SessionConn
	clock   clock.Clock
}

// Get value from mysql session
func (p *connection) Get(key string) string {
	switch strings.ToLower(key) {
	case "username":
		return p.conn.UserName
	default:
		return p.conn.Data.Get(key)
	}
}

// Get value from mysql session
func (p *connection) GetValues() map[string][]string {
	return p.conn.Data
}

// Set value in mysql session.
// it is temp value in map.
func (p *connection) Set(key, value string) error {
	switch strings.ToLower(key) {
	case "username":
		p.conn.UserName = value
	default:
		p.conn.Data.Set(key, value)
	}
	return nil
}

func (p *connection) Add(key, value string) error {
	switch strings.ToLower(key) {
	case "username":
		p.conn.UserName = value
	default:
		p.conn.Data.Add(key, value)
	}
	return nil
}

func (p *connection) Del(key string) {
	p.conn.Data.Del(key)
}

func (p *connection) Has(key string) bool {
	return p.conn.Data.Has(key)
}

// Reset clear all values in mysql session
func (p *connection) Reset() error {
	p.conn.UserName = ""
	p.conn.Data = make(url.Values)
	return nil
}

// Sid get session id of this mysql session store
func (p *connection) Sid() string {
	return p.conn.Sid
}

func (p *connection) Update(w http.ResponseWriter) error {
	p.conn.UpdatedAt = p.clock.Now().Unix()
	return p.session.Update(context.Background(), p.conn)
}

func (p *connection) CreatedAt() time.Time {
	return time.Unix(p.conn.CreatedAt, 0)
}

// }}}
