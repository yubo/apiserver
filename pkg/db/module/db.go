package db

import (
	"context"

	"github.com/yubo/apiserver/pkg/db"
	"github.com/yubo/apiserver/pkg/options"
	"github.com/yubo/golib/proc"
)

const (
	moduleName = "db"
)

type module struct {
	name string
	db   db.DB
}

var (
	_module = &module{name: moduleName}
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
func (p *module) init(ctx context.Context) (err error) {
	cf := newConfig()
	if err := proc.ReadConfig(p.name, cf); err != nil {
		return err
	}

	if p.db, err = db.NewDB(ctx, cf); err != nil {
		return err
	}
	options.WithDB(ctx, p.db)

	return nil
}

func (p *module) stop(ctx context.Context) error {
	return p.db.Close()
}

func newConfig() *db.Config {
	return &db.Config{}
}

func Register() {
	proc.RegisterHooks(hookOps)

	proc.AddConfig(moduleName, newConfig(), proc.WithConfigGroup("db"))
}
