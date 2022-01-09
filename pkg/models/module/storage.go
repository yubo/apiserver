package models

import (
	"context"

	"github.com/yubo/apiserver/pkg/db"
	"github.com/yubo/apiserver/pkg/models"
	"github.com/yubo/apiserver/pkg/options"
	"github.com/yubo/apiserver/pkg/storage"
	dbstore "github.com/yubo/apiserver/pkg/storage/db"
	"github.com/yubo/golib/proc"
)

const (
	moduleName = "models"
)

type module struct {
	name    string
	db      db.DB
	storage storage.Interface
}

var (
	_module = &module{name: moduleName}
	hookOps = []proc.HookOps{{
		Hook:        _module.init,
		Owner:       moduleName,
		HookNum:     proc.ACTION_START,
		Priority:    proc.PRI_SYS_INIT,
		SubPriority: options.PRI_M_DB + 1,
	}}
)

// Because some configuration may be stored in the database,
// set the db.connect into sys.db.prestart
func (p *module) init(ctx context.Context) (err error) {
	p.db = options.DBMustFrom(ctx, "")
	p.storage = dbstore.New(p.db)

	models.SetStorage(p.storage, "")

	if err := models.Prepare(); err != nil {
		return err
	}
	return nil
}

func Register() {
	proc.RegisterHooks(hookOps)
}
