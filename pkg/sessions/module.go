package sessions

import (
	"context"
	"net/http"
	"strings"

	"github.com/yubo/apiserver/pkg/proc"
	v1 "github.com/yubo/apiserver/pkg/proc/api/v1"
	"github.com/yubo/golib/api"
	"github.com/yubo/golib/util"
	"github.com/yubo/golib/util/clock"
	"github.com/yubo/golib/util/errors"
)

const moduleName = "session"

var (
	factories = map[string]StoreFactory{}
)

func Register(opts ...proc.ModuleOption) {
	o := &proc.ModuleOptions{
		Proc: proc.DefaultProcess,
	}
	for _, v := range opts {
		v(o)
	}

	module := &module{name: moduleName}
	hookOps := []v1.HookOps{{
		Hook:        module.init,
		Owner:       moduleName,
		HookNum:     v1.ACTION_START,
		Priority:    v1.PRI_SYS_INIT,
		SubPriority: v1.PRI_M_AUTHN,
	}}

	o.Proc.RegisterHooks(hookOps)
	o.Proc.AddConfig(moduleName, newConfig(), proc.WithConfigGroup("session"))
}

type StoreFactory func(ctx context.Context, option *Options) (Store, error)

func RegisterStore(name string, f StoreFactory) error {
	if _, ok := factories[name]; ok {
		return errors.Errorf("session store %s already registered", name)
	}
	factories[name] = f
	return nil
}

func newConfig() *config {
	return &config{
		Path:   "/",
		MaxAge: api.NewDuration("720h"),
	}
}

type config struct {
	Path   string `json:"path"`
	Domain string `json:"domain"`
	// MaxAge=0 means no 'Max-Age' attribute specified.
	// MaxAge<0 means delete cookie now, equivalently 'Max-Age: 0'.
	// MaxAge>0 means Max-Age attribute present and given in seconds.
	MaxAge   api.Duration `json:"maxAge"`
	Secure   bool         `json:"secure"`
	HttpOnly bool         `json:"httpOnly"`
	// rfc-draft to preventing CSRF: https://tools.ietf.org/html/draft-west-first-party-cookies-07
	//   refer: https://godoc.org/net/http
	//          https://www.sjoerdlangkemper.nl/2016/04/14/preventing-csrf-with-samesite-cookie-attribute/
	SameSite string `json:"sameSite"`

	Name     string   `json:"name"`
	Store    string   `json:"store"`
	KeyPairs [][]byte `json:"keyPairs"`
}

func (p *config) Options(c clock.WithTicker) *Options {
	if util.IsNil(c) {
		c = clock.RealClock{}
	}

	opts := &Options{
		Name:     p.Name,
		Clock:    c,
		Path:     p.Path,
		Domain:   p.Domain,
		MaxAge:   int(p.MaxAge.Seconds()),
		Secure:   p.Secure,
		HttpOnly: p.HttpOnly,
		KeyPairs: p.KeyPairs,
	}

	switch strings.ToLower(p.SameSite) {
	case "lax":
		opts.SameSite = http.SameSiteLaxMode
	case "strict":
		opts.SameSite = http.SameSiteStrictMode
	case "none":
		opts.SameSite = http.SameSiteNoneMode
	default:
		opts.SameSite = http.SameSiteDefaultMode
	}

	return opts
}

func (p *config) Validate() error {
	if p == nil {
		return nil
	}

	return nil
}

type module struct {
	name string
}

// Because some configuration may be stored in the database,
// set the db.connect into sys.db.prestart
func (p *module) init(ctx context.Context) error {
	cf := newConfig()
	if err := proc.ReadConfig(p.name, cf); err != nil {
		return err
	}

	factory, ok := factories[cf.Store]
	if !ok {
		return errors.Errorf("session store %s does not exist", cf.Store)
	}

	store, err := factory(ctx, cf.Options(nil))
	if err != nil {
		return err
	}

	SetStore(store)

	return nil
}
