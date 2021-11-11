package db

import (
	"context"

	"github.com/yubo/apiserver/pkg/db"
	"github.com/yubo/apiserver/pkg/options"
	"github.com/yubo/golib/configer"
	"github.com/yubo/golib/proc"
)

const (
	moduleName = "db"
)

type dbModule struct {
	name string
	db   db.DB
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

	cf := &db.Config{}
	if err := configer.Read(p.name, cf); err != nil {
		return err
	}

	if p.db, err = db.NewDB(ctx, cf); err != nil {
		return err
	}
	options.WithDB(ctx, p.db)

	return nil
}

func (p *dbModule) stop(ctx context.Context) error {
	return p.db.Close()
}

func Register() {
	proc.RegisterHooks(hookOps)

	cf := &db.Config{}
	proc.RegisterFlags("db", "db", cf, configer.WithTags(cf.Tags))
}
