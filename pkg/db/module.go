package db

import (
	"context"

	"github.com/yubo/apiserver/components/dbus"
	"github.com/yubo/apiserver/pkg/db/api"
	"github.com/yubo/apiserver/pkg/proc"
	v1 "github.com/yubo/apiserver/pkg/proc/api/v1"
	"github.com/yubo/golib/orm"
)

const moduleName = "db"

func Register(opts ...proc.ModuleOption) {
	o := &proc.ModuleOptions{
		Proc: proc.DefaultProcess,
	}
	for _, opt := range opts {
		opt(o)
	}

	module := &module{
		name: moduleName,
		proc: o.Proc,
	}
	hookOps := []v1.HookOps{{
		Hook:        module.init,
		Owner:       moduleName,
		HookNum:     v1.ACTION_START,
		Priority:    v1.PRI_SYS_INIT,
		SubPriority: v1.PRI_M_DB,
	}, {

		Hook:        module.stop,
		Owner:       moduleName,
		HookNum:     v1.ACTION_STOP,
		Priority:    v1.PRI_SYS_INIT,
		SubPriority: v1.PRI_M_DB,
	}}

	o.Proc.RegisterHooks(hookOps)
	o.Proc.AddConfig(moduleName, newConfig(), proc.WithConfigGroup("DB"))
}

type module struct {
	name string
	db   api.DB
	proc *proc.Process
}

// Because some configuration may be stored in the database,
// set the db.connect into sys.db.prestart
func (p *module) init(ctx context.Context) (err error) {
	cf := newConfig()
	if err := p.proc.ReadConfig(p.name, cf); err != nil {
		return err
	}

	if p.db, err = NewDB(ctx, cf); err != nil {
		return err
	}
	if cf.Debug {
		orm.DEBUG = true
	}
	dbus.RegisterDB(p.db)

	if err = autoMigrate(ctx, p.db); err != nil {
		return err
	}

	return nil
}

func (p *module) stop(ctx context.Context) error {
	return p.db.Close()
}

func newConfig() *Config {
	return &Config{}
}

func NewConfig(driver, dsn string) *Config {
	c := &Config{}
	c.Driver = driver
	c.Dsn = dsn
	return c
}
