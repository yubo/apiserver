package register

import (
	"context"

	"github.com/yubo/apiserver/components/dbus"
	"github.com/yubo/apiserver/pkg/db"
	"github.com/yubo/apiserver/pkg/proc"
	v1 "github.com/yubo/apiserver/pkg/proc/api/v1"
	"github.com/yubo/golib/orm"
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
	hookOps = []v1.HookOps{{
		Hook:        _module.init,
		Owner:       moduleName,
		HookNum:     v1.ACTION_START,
		Priority:    v1.PRI_SYS_INIT,
		SubPriority: v1.PRI_M_DB,
	}, {
		Hook:        _module.stop,
		Owner:       moduleName,
		HookNum:     v1.ACTION_STOP,
		Priority:    v1.PRI_SYS_INIT,
		SubPriority: v1.PRI_M_DB,
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
	if cf.Debug {
		orm.DEBUG = true
	}
	dbus.RegisterDB(p.db)

	return nil
}

func (p *module) stop(ctx context.Context) error {
	return p.db.Close()
}

func newConfig() *db.Config {
	return &db.Config{}
}

func init() {
	proc.RegisterHooks(hookOps)

	proc.AddConfig(moduleName, newConfig(), proc.WithConfigGroup("DB"))
}
