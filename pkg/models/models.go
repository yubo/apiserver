package models

import (
	"context"
	"fmt"

	"github.com/yubo/apiserver/pkg/db"
	"github.com/yubo/apiserver/pkg/options"
	"github.com/yubo/apiserver/pkg/storage"
	dbstore "github.com/yubo/apiserver/pkg/storage/db"
	"github.com/yubo/apiserver/pkg/storage/etcd"
	"github.com/yubo/apiserver/pkg/storage/file"
	"github.com/yubo/apiserver/pkg/storage/mem"
	"github.com/yubo/golib/orm"
	"github.com/yubo/golib/proc"
	"github.com/yubo/golib/util/errors"
)

const (
	moduleName = "models"
)

type module struct {
	db.DB
	name   string
	config *Config
	kv     storage.KV

	registry map[string]Model
	models   []Model
}

type Config struct {
	Storage     string `json:"storage" description:"kv storage type, db"`
	DBName      string `json:"dbName" flag:"models-db-name" description:"the database name of db.databases"`
	AutoMigrate bool   `json:"autoMigrate" flag:"models-automigrate" description:"auto migrate"`
	TablePrefix string `json:"tablePrefix" flag:"models-table-prefix" description:"table name prefix of the database"`
}

func newConfig() *Config {
	return &Config{
		Storage:     "db",
		DBName:      "",
		AutoMigrate: true,
	}
}

var (
	_module = &module{
		name:     moduleName,
		registry: make(map[string]Model),
	}
	hookOps = []proc.HookOps{{
		Hook:        _module.init,
		Owner:       moduleName,
		HookNum:     proc.ACTION_START,
		Priority:    proc.PRI_SYS_INIT,
		SubPriority: options.PRI_M_DB + 1,
	}, {
		// after moduels start
		// before services start
		Hook:        _module.preStart,
		Owner:       moduleName,
		HookNum:     proc.ACTION_START,
		Priority:    proc.PRI_SYS_PRESTART,
		SubPriority: options.PRI_M_DB + 1,
	}}
)

// Because some configuration may be stored in the database,
// set the db.connect into sys.db.prestart
func (p *module) init(ctx context.Context) (err error) {
	cf := newConfig()
	if err := proc.ReadConfig(p.name, cf); err != nil {
		return err
	}

	p.config = cf

	switch cf.Storage {
	case "db":
		var ok bool
		p.DB, ok = options.DBFrom(ctx, cf.DBName)
		if !ok {
			return fmt.Errorf("unable to get db[%s] from context", cf.DBName)
		}

		p.kv = dbstore.New(p.DB)
	case "etcd":
		p.kv = etcd.New()
	case "file":
		p.kv = file.New()
	case "mem":
		p.kv = mem.New()

	default:
		return fmt.Errorf("unsupported storage %s", cf.Storage)
	}

	return nil
}

func (p *module) preStart(ctx context.Context) error {
	// automigrate
	if p.config.Storage == "db" && p.config.AutoMigrate {
		var errs []error
		for _, m := range p.models {
			if err := p.AutoMigrate(m.NewObj(), orm.WithTable(m.Name())); err != nil {
				errs = append(errs, err)
			}
		}
		if err := errors.NewAggregate(errs); err != nil {
			return err
		}
	}
	return nil
}

// Register: register models
func (p *module) Register(ms ...Model) {
	for _, m := range ms {
		name := m.Name()
		if _, ok := p.registry[name]; ok {
			panic(fmt.Sprintf("%s has already been registered", name))
		}

		p.registry[name] = m
		p.models = append(p.models, m)
	}
}

func (p *module) NewStore(kind string) Store {
	if _, ok := p.registry[kind]; !ok {
		panic(fmt.Sprintf("model %s that has not been registered", kind))
	}

	if p.kv == nil {
		panic("storage that has not been set")
	}

	return Store{
		kv:       p.kv,
		resource: kind,
	}
}

func RegisterModule() {
	proc.RegisterHooks(hookOps)
	proc.AddConfig("models", newConfig(), proc.WithConfigGroup("models"))
}

func Register(ms ...Model) {
	_module.Register(ms...)
}

func NewStore(kind string) Store {
	return _module.NewStore(kind)
}

func DB() orm.DB {
	return _module.DB
}
