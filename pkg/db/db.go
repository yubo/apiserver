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

type config struct {
	Driver string `json:"driver" flag:"db-driver" default:"mysql" description:"default: mysql"`
	Dsn    string `json:"dsn" flag:"db-dsn" description:"db is disabled when empty dsn"`
}

func (p config) String() string {
	return util.Prettify(p)
}

func (p *config) Validate() error {
	if p.Dsn == "" {
		return nil
	}
	if p.Driver == "" {
		p.Driver = "mysql"
	}
	return nil
}

type dbModule struct {
	config *config
	name   string
	db     orm.DB
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
func (p *dbModule) init(ctx context.Context) (err error) {
	configer := proc.ConfigerMustFrom(ctx)
	p.ctx, p.cancel = context.WithCancel(ctx)

	cf := &config{}
	if err := configer.Read(p.name, cf); err != nil {
		return err
	}

	p.config = cf

	// db
	if cf.Driver != "" && cf.Dsn != "" {
		if p.db, err = orm.Open(cf.Driver, cf.Dsn, orm.WithContext(p.ctx)); err != nil {
			return err
		}
		options.WithDB(ctx, p.db)
	}

	return nil
}

func (p *dbModule) stop(ctx context.Context) error {
	p.cancel()
	return nil
}

func Register() {
	proc.RegisterHooks(hookOps)
	proc.RegisterFlags("db", "db", &config{})
}
