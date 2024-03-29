package sessions

import (
	"context"
	"log"
	"net/http"

	"github.com/gorilla/sessions"
	gsessions "github.com/gorilla/sessions"
	"github.com/yubo/golib/util/clock"
)

const (
	errorFormat = "[sessions] ERROR! %s\n"
	UserInfoKey = "userInfo"
)

type key int

const (
	SessionKey key = iota
	ManySessionKey
)

func withSession(parent context.Context, s Session) context.Context {
	return context.WithValue(parent, SessionKey, s)
}

func SessionFrom(ctx context.Context) (Session, bool) {
	s, ok := ctx.Value(SessionKey).(Session)
	return s, ok
}

func SessionMustFrom(ctx context.Context) Session {
	s, ok := ctx.Value(SessionKey).(Session)
	if !ok {
		panic("session does not exist")
	}
	return s
}

// shortcut to get session
func Default(ctx context.Context) Session {
	return SessionMustFrom(ctx)
}

func withManySession(parent context.Context, v map[string]Session) context.Context {
	return context.WithValue(parent, ManySessionKey, v)
}

func ManySessionFrom(ctx context.Context) (map[string]Session, bool) {
	s, ok := ctx.Value(ManySessionKey).(map[string]Session)
	return s, ok
}

func ManySessionMustFrom(ctx context.Context) map[string]Session {
	s, ok := ctx.Value(ManySessionKey).(map[string]Session)
	if !ok {
		panic("session does not exist")
	}
	return s
}

// shortcut to get session with given name
func DefaultMany(ctx context.Context, name string) Session {
	return ManySessionMustFrom(ctx)[name]
}

type Store interface {
	Name() string
	Type() string
	sessions.Store
}

// Wraps thinly gorilla-session methods.
// Session stores the values and optional configuration for a session.
type Session interface {
	// ID of the session, generated by stores. It should not be used for user data.
	ID() string
	// Get returns the session value associated to the given key.
	Get(key interface{}) interface{}
	// Set sets the session value associated to the given key.
	Set(key interface{}, val interface{})
	// Delete removes the session value associated to the given key.
	Delete(key interface{})
	// Clear deletes all values in the session.
	Clear()
	// AddFlash adds a flash message to the session.
	// A single variadic argument is accepted, and it is optional: it defines the flash key.
	// If not defined "_flash" is used by default.
	AddFlash(value interface{}, vars ...string)
	// Flashes returns a slice of flash messages from the session.
	// A single variadic argument is accepted, and it is optional: it defines the flash key.
	// If not defined "_flash" is used by default.
	Flashes(vars ...string) []interface{}
	// Options sets configuration for a session.
	Options(Options)
	// Save saves all sessions used during the current request.
	Save() error
}

type Options struct {
	Clock clock.WithTicker

	Name     string
	KeyPairs [][]byte

	Path   string
	Domain string
	// MaxAge=0 means no 'Max-Age' attribute specified.
	// MaxAge<0 means delete cookie now, equivalently 'Max-Age: 0'.
	// MaxAge>0 means Max-Age attribute present and given in seconds.
	MaxAge   int
	Secure   bool
	HttpOnly bool
	// rfc-draft to preventing CSRF: https://tools.ietf.org/html/draft-west-first-party-cookies-07
	//   refer: https://godoc.org/net/http
	//          https://www.sjoerdlangkemper.nl/2016/04/14/preventing-csrf-with-samesite-cookie-attribute/
	SameSite http.SameSite
}

func (options *Options) ToGorillaOptions() *gsessions.Options {
	return &gsessions.Options{
		Path:     options.Path,
		Domain:   options.Domain,
		MaxAge:   options.MaxAge,
		Secure:   options.Secure,
		HttpOnly: options.HttpOnly,
		SameSite: options.SameSite,
	}
}

type session struct {
	name    string
	request *http.Request
	store   Store
	session *sessions.Session
	written bool
	writer  http.ResponseWriter
}

func (s *session) ID() string {
	return s.Session().ID
}

func (s *session) Get(key interface{}) interface{} {
	return s.Session().Values[key]
}

func (s *session) Set(key interface{}, val interface{}) {
	s.Session().Values[key] = val
	s.written = true
}

func (s *session) Delete(key interface{}) {
	delete(s.Session().Values, key)
	s.written = true
}

func (s *session) Clear() {
	for key := range s.Session().Values {
		s.Delete(key)
	}
}

func (s *session) AddFlash(value interface{}, vars ...string) {
	s.Session().AddFlash(value, vars...)
	s.written = true
}

func (s *session) Flashes(vars ...string) []interface{} {
	s.written = true
	return s.Session().Flashes(vars...)
}

func (s *session) Options(options Options) {
	s.written = true
	s.Session().Options = options.ToGorillaOptions()
}

func (s *session) Save() error {
	if s.Written() {
		e := s.Session().Save(s.request, s.writer)
		if e == nil {
			s.written = false
		}
		return e
	}
	return nil
}

func (s *session) Session() *sessions.Session {
	if s.session == nil {
		var err error
		s.session, err = s.store.Get(s.request, s.name)
		if err != nil {
			log.Printf(errorFormat, err)
		}
	}
	return s.session
}

func (s *session) Written() bool {
	return s.written
}
