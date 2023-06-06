package models

import (
	"context"
	"fmt"

	"github.com/yubo/apiserver/components/dbus"
	"github.com/yubo/apiserver/pkg/db"
	"github.com/yubo/apiserver/pkg/proc"
	v1 "github.com/yubo/apiserver/pkg/proc/api/v1"
	"github.com/yubo/apiserver/pkg/storage"
	dbstore "github.com/yubo/apiserver/pkg/storage/db"
	"github.com/yubo/apiserver/pkg/storage/etcd"
	"github.com/yubo/apiserver/pkg/storage/file"
	"github.com/yubo/apiserver/pkg/storage/mem"
	"github.com/yubo/golib/orm"
	"github.com/yubo/golib/util/errors"
)

const (
	moduleName = "models"
)

type module struct {
	db.DB
	name   string
	config *Config
	store  storage.Store

	registry map[string]Model
	models   []Model
}

type Config struct {
	Storage     string `json:"storage" description:"storage type, db"`
	DBName      string `json:"dbName" description:"the database name of db.databases"`
	AutoMigrate bool   `json:"autoMigrate" description:"auto migrate"`
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
	hookOps = []v1.HookOps{{
		Hook:        _module.init,
		Owner:       moduleName,
		HookNum:     v1.ACTION_START,
		Priority:    v1.PRI_SYS_INIT,
		SubPriority: v1.PRI_M_DB + 1,
	}, {
		// after moduels start
		// before services start
		Hook:        _module.preStart,
		Owner:       moduleName,
		HookNum:     v1.ACTION_START,
		Priority:    v1.PRI_SYS_PRESTART,
		SubPriority: v1.PRI_M_DB + 1,
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
		if db := dbus.DB().GetDB(cf.DBName); db == nil {
			return fmt.Errorf("unable to get db[%s] from context", cf.DBName)
		} else {
			p.store = dbstore.New(db)
			p.DB = db
		}
	case "etcd":
		p.store = etcd.New()
	case "file":
		p.store = file.New()
	case "mem":
		p.store = mem.New()

	default:
		return fmt.Errorf("unsupported storage %s", cf.Storage)
	}

	return nil
}

func (p *module) preStart(ctx context.Context) error {
	// automigrate
	if p.config.AutoMigrate && p.DB != nil {
		var errs []error
		for _, m := range p.models {
			if err := p.AutoMigrate(ctx, m.NewObj(), orm.WithTable(m.Name())); err != nil {
				errs = append(errs, err)
			}
		}
		return errors.NewAggregate(errs)
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

func (p *module) NewModelStore(kind string) ModelStore {
	if _, ok := p.registry[kind]; !ok {
		panic(fmt.Sprintf("model %s that has not been registered", kind))
	}

	if p.store == nil {
		panic("storage that has not been set")
	}

	return ModelStore{
		store:    p.store,
		resource: kind,
	}
}

// for test
func NewModels(s storage.Store) Models {
	return &module{
		store:    s,
		registry: map[string]Model{},
	}
}

func RegisterModule() {
	proc.RegisterHooks(hookOps)
	proc.AddConfig("models", newConfig(), proc.WithConfigGroup("models"))
}

func Register(ms ...Model) {
	_module.Register(ms...)
}

func NewModelStore(kind string) ModelStore {
	return _module.NewModelStore(kind)
}

func DB() orm.DB {
	if _module.DB == nil {
		panic("invalid db")
	}
	return _module.DB
}
