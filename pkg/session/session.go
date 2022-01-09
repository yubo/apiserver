package session

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/yubo/golib/api/errors"
	"github.com/yubo/golib/util"
	"github.com/yubo/golib/util/clock"
	"k8s.io/klog/v2"
)

type storage interface {
	all() int
	get(sid string) (*sessionConn, error)
	insert(*sessionConn) error
	del(sid string) error
	update(*sessionConn) error
}

func second2duration(second int, def time.Duration) time.Duration {
	if second == 0 {
		return def
	}

	return time.Duration(second) * time.Second

}

type SessionManager interface {
	Start(w http.ResponseWriter, r *http.Request) (store Session, err error)
	StopGC()
	Destroy(w http.ResponseWriter, r *http.Request) error
	Get(sid string) (Session, error)
	All() int
}

type Session interface {
	Set(key, value string) error
	Get(key string) string
	CreatedAt() time.Time
	Delete(key string) error
	Reset() error
	Sid() string
	Update(w http.ResponseWriter) error
}

func StartSession(cf *Config, optsInput ...Option) (SessionManager, error) {
	opts := Options{}

	for _, opt := range optsInput {
		opt(&opts)
	}

	if opts.ctx == nil {
		opts.ctx, opts.cancel = context.WithCancel(context.Background())
	}

	if opts.clock == nil {
		opts.clock = clock.RealClock{}
	}

	var storage storage
	var err error
	if cf.Storage == "mem" {
		storage, err = newMemStorage(cf, &opts)
	} else {
		storage, err = newDbStorage(cf, &opts)
	}

	if err != nil {
		return nil, err
	}

	return &sessionManager{
		storage: storage,
		config:  cf,
		Options: opts,
	}, nil
}

type sessionConn struct {
	Sid        string `sql:"sid,where,primary_key"`
	UserName   string
	Data       map[string]string
	CookieName string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type sessionManager struct {
	storage
	Options
	config *Config
}

// SessionStart generate or read the session id from http request.
// if session id exists, return SessionStore with this id.
func (p *sessionManager) Start(w http.ResponseWriter, r *http.Request) (sess Session, err error) {
	var sid string

	if sid, err = p.getSid(r); err != nil {
		return
	}

	if sid != "" {
		if sess, err := p.getSessionStore(sid, false); err == nil {
			return sess, nil
		}
	}

	// Generate a new session
	sid = util.RandString(p.config.SidLength)

	sess, err = p.getSessionStore(sid, true)
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

func (p *sessionManager) StopGC() {
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
	p.del(sid)

	cookie = &http.Cookie{Name: p.config.CookieName,
		Path:     "/",
		HttpOnly: p.config.HttpOnly,
		Expires:  p.clock.Now(),
		MaxAge:   -1}

	http.SetCookie(w, cookie)
	return nil
}

func (p *sessionManager) Get(sid string) (Session, error) {
	return p.getSessionStore(sid, true)
}

func (p *sessionManager) Exist(sid string) bool {
	_, err := p.get(sid)
	return !errors.IsNotFound(err)
}

// All count values in mysql session
func (p *sessionManager) All() int {
	return p.all()
}

func (p *sessionManager) getSid(r *http.Request) (sid string, err error) {
	var cookie *http.Cookie

	cookie, err = r.Cookie(p.config.CookieName)
	if err != nil || cookie.Value == "" {
		return sid, nil
	}

	return url.QueryUnescape(cookie.Value)
}

func (p *sessionManager) getSessionStore(sid string, create bool) (Session, error) {
	sc, err := p.get(sid)
	if errors.IsNotFound(err) && create {
		ts := p.clock.Now()
		sc = &sessionConn{
			Sid:        sid,
			CookieName: p.config.CookieName,
			CreatedAt:  ts,
			UpdatedAt:  ts,
			Data:       make(map[string]string),
		}
		err = p.insert(sc)
	}
	if err != nil {
		return nil, err
	}
	return &session{manager: p, conn: sc}, nil
}

// session mysql session store
type session struct {
	sync.RWMutex
	conn    *sessionConn
	manager *sessionManager
}

// Set value in mysql session.
// it is temp value in map.
func (p *session) Set(key, value string) error {
	p.Lock()
	defer p.Unlock()
	klog.Infof("entering set key %s v %s", key, value)

	switch strings.ToLower(key) {
	case "username":
		p.conn.UserName = value
	default:
		p.conn.Data[key] = value
	}
	return nil
}

// Get value from mysql session
func (p *session) Get(key string) string {
	p.RLock()
	defer p.RUnlock()
	klog.Infof("entering get key %s", key)

	switch strings.ToLower(key) {
	case "username":
		return p.conn.UserName
	default:
		return p.conn.Data[key]
	}
}

func (p *session) CreatedAt() time.Time {
	return p.conn.CreatedAt
}

// Delete value in mysql session
func (p *session) Delete(key string) error {
	p.Lock()
	defer p.Unlock()
	klog.Infof("entering delete")
	delete(p.conn.Data, key)
	return nil
}

// Reset clear all values in mysql session
func (p *session) Reset() error {
	p.Lock()
	defer p.Unlock()
	klog.Infof("entering reset")

	p.conn.UserName = ""
	p.conn.Data = make(map[string]string)
	return nil
}

// Sid get session id of this mysql session store
func (p *session) Sid() string {
	return p.conn.Sid
}

func (p *session) Update(w http.ResponseWriter) error {
	klog.Infof("entering update")
	p.conn.UpdatedAt = p.manager.clock.Now()
	return p.manager.update(p.conn)
}
