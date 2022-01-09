package models

import (
	"fmt"
	"sync"

	"github.com/yubo/apiserver/pkg/storage"
	"github.com/yubo/golib/runtime"
	"github.com/yubo/golib/util/errors"

	_ "github.com/yubo/apiserver/pkg/db/register"
)

var (
	defaultModels = NewModels()
)

func SetStorage(s storage.Interface, prefix string) {
	defaultModels.SetStorage(s, prefix)
}

func Register(ms ...Model) {
	for _, m := range ms {
		defaultModels.Register(m)
	}
}

func Prepare() error {
	return defaultModels.Prepare()
}

func AutoMigrate(name string, obj runtime.Object) error {
	return defaultModels.AutoMigrate(name, obj)
}

func NewStore(kind string) Store {
	return defaultModels.NewStore(kind)
}

func NewModels() *models {
	return &models{
		registry: map[string]Model{},
		models:   []Model{},
	}
}

type models struct {
	storage  storage.Interface
	prefix   string
	registry map[string]Model
	models   []Model
	prepare  sync.Once
}

func (p *models) SetStorage(s storage.Interface, prefix string) {
	p.storage = s
	p.prefix = prefix
}

func (p *models) Register(ms ...Model) {
	for _, m := range ms {
		name := m.Name()
		if _, ok := p.registry[name]; ok {
			panic(fmt.Sprintf("%s has already been registered", name))
		}

		p.registry[name] = m
		p.models = append(p.models, m)
	}
}

func (p *models) NewStore(kind string) Store {
	if _, ok := p.registry[kind]; !ok {
		panic(fmt.Sprintf("model %s that has not been registered", kind))
	}

	if p.storage == nil {
		panic("storage that has not been set")
	}

	return Store{
		s:        p.storage,
		prefix:   p.prefix,
		resource: kind,
	}
}

// Prepare: check/autoMigrate all models are available
func (p *models) Prepare() (err error) {
	p.prepare.Do(func() {
		if p.storage == nil {
			err = fmt.Errorf("storage that has not been set")
			return
		}

		var errs []error
		for _, m := range p.models {
			if err := p.AutoMigrate(m.Name(), m.NewObj()); err != nil {
				errs = append(errs, err)
			}
		}

		err = errors.NewAggregate(errs)
	})

	return
}

func (p *models) AutoMigrate(name string, obj runtime.Object) error {
	return p.storage.AutoMigrate(p.prefix+name, obj)
}
