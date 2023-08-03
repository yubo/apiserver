package orm

import (
	"context"
	"encoding/base32"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/securecookie"
	gsessions "github.com/gorilla/sessions"
	"github.com/yubo/apiserver/components/dbus"
	"github.com/yubo/apiserver/pkg/db"
	"github.com/yubo/apiserver/pkg/proc"
	"github.com/yubo/apiserver/pkg/sessions"
	"github.com/yubo/golib/api"
	"github.com/yubo/golib/orm"
	"github.com/yubo/golib/util"
	"github.com/yubo/golib/util/clock"
	"k8s.io/klog/v2"
)

const (
	storeType  = "orm"
	moduleName = "session.orm"
)

type Config struct {
	Ctx     context.Context   `json:"-"`
	Orm     orm.DB            `json:"-"`
	Options *sessions.Options `json:"-"`

	DB              *db.Config   `json:"db"`
	TableName       string       `json:"tableName"`
	SkipCreateTable bool         `json:"skipCreateTable"`
	CleanupInterval api.Duration `json:"cleanupInterval"`
}

func newConfig() *Config {
	return &Config{
		Ctx:             context.Background(),
		TableName:       "session",
		SkipCreateTable: false,
		CleanupInterval: api.NewDuration("1h"),
	}
}

type OrmSession struct {
	ID        *string `sql:"unique,where"`
	Data      *string `sql:"type=text"`
	CreatedAt *time.Time
	UpdatedAt *time.Time
	ExpiresAt *time.Time `sql:"index"`
}

func NewStore(config *Config) (sessions.Store, error) {
	p := &store{
		name:      config.Options.Name,
		db:        config.Orm,
		clock:     config.Options.Clock,
		tableName: config.TableName,
		Options:   config.Options.ToGorillaOptions(),
		Codecs:    securecookie.CodecsFromPairs(config.Options.KeyPairs...),
	}

	if p.tableName == "" {
		return nil, fmt.Errorf("invalidate tableName")
	}

	if p.db == nil || p.db.SqlDB().Ping() != nil {
		return nil, fmt.Errorf("invalidate orm")
	}

	if p.clock == nil {
		p.clock = clock.RealClock{}
	}

	if !config.SkipCreateTable {
		if err := p.db.AutoMigrate(context.Background(), &OrmSession{}, orm.WithTable(p.tableName)); err != nil {
			return nil, err
		}
	}

	if interval := config.CleanupInterval.Duration; interval.Nanoseconds() > 0 {
		p.PeriodicCleanup(config.Ctx, interval)
	}

	p.MaxAge(p.Options.MaxAge)

	return p, nil
}

type store struct {
	name      string
	db        orm.DB
	clock     clock.WithTicker
	tableName string
	Codecs    []securecookie.Codec
	Options   *gsessions.Options
}

func (s *store) Name() string {
	return s.name
}
func (s *store) Type() string {
	return storeType
}

// Get returns a session for the given name after adding it to the registry.
func (s *store) Get(r *http.Request, name string) (*gsessions.Session, error) {
	return gsessions.GetRegistry(r).Get(s, name)
}

// New creates a session with name without adding it to the registry.
func (p *store) New(req *http.Request, name string) (*gsessions.Session, error) {
	session := gsessions.NewSession(p, name)
	opts := *p.Options
	session.Options = &opts
	session.IsNew = true

	// try fetch from db if there is a cookie
	var err error
	if cookie, errCookie := req.Cookie(name); errCookie == nil {
		err = securecookie.DecodeMulti(name, cookie.Value, &session.ID, p.Codecs...)
		if err == nil {
			err = p.load(req.Context(), session)
			if err == nil {
				session.IsNew = false
			}
		}
	}

	return session, err
}

// Save session and set cookie header
func (p *store) Save(req *http.Request, w http.ResponseWriter, session *gsessions.Session) error {
	ctx := req.Context()

	// delete if max age is < 0
	if session.Options.MaxAge < 0 {
		if err := p.db.ExecNumErr(ctx, "delete from `"+p.tableName+"` where id=?", session.ID); err != nil {
			return err
		}
		http.SetCookie(w, gsessions.NewCookie(session.Name(), "", session.Options))
		return nil
	}

	if err := p.save(ctx, session); err != nil {
		return err
	}

	encoded, err := securecookie.EncodeMulti(session.Name(), session.ID, p.Codecs...)
	if err != nil {
		return err
	}
	http.SetCookie(w, gsessions.NewCookie(session.Name(), encoded, session.Options))
	return nil
}

func (p *store) save(ctx context.Context, session *gsessions.Session) error {
	data, err := securecookie.EncodeMulti(session.Name(), session.Values, p.Codecs...)
	if err != nil {
		return err
	}
	now := p.clock.Now()
	expire := now.Add(time.Second * time.Duration(session.Options.MaxAge))

	if session.IsNew {
		// generate random session ID key suitable for storage in the db
		session.ID = strings.TrimRight(base32.StdEncoding.EncodeToString(
			securecookie.GenerateRandomKey(32)), "=")
		return p.db.Insert(ctx, &OrmSession{
			ID:        &session.ID,
			Data:      &data,
			CreatedAt: &now,
			UpdatedAt: &now,
			ExpiresAt: &expire,
		}, orm.WithTable(p.tableName))
	}

	return p.db.Update(ctx, &OrmSession{
		ID:        &session.ID,
		Data:      &data,
		UpdatedAt: &now,
		ExpiresAt: &expire,
	}, orm.WithTable(p.tableName))
}

// load query and decodes its content into session.Values.
func (p *store) load(ctx context.Context, session *gsessions.Session) error {
	s := &OrmSession{}
	if err := p.db.Query(ctx, "select * from `"+p.tableName+"` where id=? and expires_at > ?", session.ID, p.clock.Now()).Row(s); err != nil {
		return err
	}

	if err := securecookie.DecodeMulti(session.Name(), util.StringValue(s.Data), &session.Values, p.Codecs...); err != nil {
		return err
	}

	return nil
}

func factory(ctx context.Context, options *sessions.Options) (sessions.Store, error) {
	cf := newConfig()
	if err := proc.ReadConfig(moduleName, cf); err != nil {
		return nil, err
	}

	cf.Ctx = ctx
	cf.Options = options

	if cf.DB != nil {
		var err error
		if cf.Orm, err = db.NewDB(ctx, cf.DB); err != nil {
			return nil, err
		}
	}

	if cf.Orm == nil {
		cf.Orm = dbus.DB().GetDB("")
	}

	if cf.Orm == nil {
		return nil, fmt.Errorf("can't find orm from context or config")
	}

	return NewStore(cf)
}

// MaxAge sets the maximum age for the store and the underlying cookie
// implementation. Individual sessions can be deleted by setting
// Options.MaxAge = -1 for that session.
func (st *store) MaxAge(age int) {
	st.Options.MaxAge = age
	for _, codec := range st.Codecs {
		if sc, ok := codec.(*securecookie.SecureCookie); ok {
			sc.MaxAge(age)
		}
	}
}

// MaxLength restricts the maximum length of new sessions to l.
// If l is 0 there is no limit to the size of a session, use with caution.
// The default is 4096 (default for securecookie)
func (st *store) MaxLength(l int) {
	for _, c := range st.Codecs {
		if codec, ok := c.(*securecookie.SecureCookie); ok {
			codec.MaxLength(l)
		}
	}
}

// Cleanup deletes expired sessions
func (p *store) Cleanup(ctx context.Context) {
	if _, err := p.db.Exec(ctx, "delete from `"+p.tableName+"` where expires_at < ?", p.clock.Now()); err != nil {
		klog.Warningf("session.Cleanup() err %s", err)
	}
}

// PeriodicCleanup runs Cleanup every interval. Close quit channel to stop.
func (p *store) PeriodicCleanup(ctx context.Context, interval time.Duration) {
	util.UntilWithTick(func() { p.Cleanup(ctx) }, p.clock.NewTicker(interval).C(), ctx.Done())
}

func init() {
	sessions.RegisterStore(storeType, factory)
}
