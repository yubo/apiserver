package db

import (
	"context"

	"github.com/yubo/apiserver/pkg/options"
	"github.com/yubo/golib/orm"
	"github.com/yubo/golib/proc"
	"github.com/yubo/golib/util"
)

const (
	moduleName = "db"
)

type Config struct {
	Driver string `json:"driver" description:"default: mysql"`
	Dsn    string `json:"dsn"`
}

func (p Config) String() string {
	return util.Prettify(p)
}

func (p *Config) Validate() error {
	if p.Dsn == "" {
		return nil
	}
	if p.Driver == "" {
		p.Driver = "mysql"
	}
	return nil
}

type dbModule struct {
	config *Config
	name   string
	db     *orm.DB
	ctx    context.Context
	cancel context.CancelFunc
}

var (
	_module = &dbModule{name: moduleName}
	hookOps = []proc.HookOps{{
		Hook:        _module.init,
		Owner:       moduleName,
		HookNum:     proc.ACTION_START,
		Priority:    proc.PRI_SYS_INIT,
		SubPriority: options.PRI_M_DB,
	}, {
		Hook:        _module.stop,
		Owner:       moduleName,
		HookNum:     proc.ACTION_STOP,
		Priority:    proc.PRI_SYS_INIT,
		SubPriority: options.PRI_M_DB,
	}}
)

// Because some configuration may be stored in the database,
// set the db.connect into sys.db.prestart
func (p *dbModule) init(ops *proc.HookOps) (err error) {
	ctx, configer := ops.ContextAndConfiger()
	p.ctx, p.cancel = context.WithCancel(ctx)

	cf := &Config{}
	if err := configer.Read(p.name, cf); err != nil {
		return err
	}

	p.config = cf

	// db
	if cf.Driver != "" && cf.Dsn != "" {
		if p.db, err = orm.DbOpenWithCtx(cf.Driver, cf.Dsn, p.ctx); err != nil {
			return err
		}
		ops.SetContext(options.WithDB(ctx, p.db))
	}

	return nil
}

func (p *dbModule) stop(ops *proc.HookOps) error {
	p.cancel()
	return nil
}

func Register() {
	proc.RegisterHooks(hookOps)
}
