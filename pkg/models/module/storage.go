package models

import (
	"context"
	"fmt"

	"github.com/yubo/apiserver/pkg/db"
	"github.com/yubo/apiserver/pkg/models"
	"github.com/yubo/apiserver/pkg/options"
	"github.com/yubo/apiserver/pkg/storage"
	dbstore "github.com/yubo/apiserver/pkg/storage/db"
	"github.com/yubo/golib/configer"
	"github.com/yubo/golib/proc"
)

const (
	moduleName = "models"
)

type module struct {
	name    string
	config  *Config
	db      db.DB
	storage storage.Interface
}

type Config struct {
	DBName      string `json:"dbName" flag:"models-db-name" description:"the database name of db.databases"`
	AutoMigrate bool   `json:"autoMigrate" flag:"models-automigrate" description:"auto migrate"`
	TablePrefix string `json:"tablePrefix" flag:"models-table-prefix" description:"table name prefix of the database"`
}

func NewConfig() *Config {
	return &Config{
		DBName:      "",
		AutoMigrate: true,
	}
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
	c := configer.ConfigerMustFrom(ctx)

	cf := NewConfig()
	if err := c.Read(p.name, cf); err != nil {
		return err
	}

	p.config = cf

	var ok bool
	p.db, ok = options.DBFrom(ctx, cf.DBName)
	if !ok {
		return fmt.Errorf("unable to get db[%s] from context", cf.DBName)
	}

	p.storage = dbstore.New(p.db)

	models.SetStorage(p.storage, cf.TablePrefix)

	if err := models.Prepare(); err != nil {
		return err
	}
	return nil
}

func Register() {
	proc.RegisterHooks(hookOps)

	cf := NewConfig()
	proc.RegisterFlags("models", "models", cf)
}
