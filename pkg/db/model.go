package db

import (
	"context"
	"fmt"

	"github.com/yubo/apiserver/pkg/db/api"
	"github.com/yubo/golib/orm"
	"github.com/yubo/golib/util/errors"
)

var (
	_registry = map[string]model{}
	_models   []model
)

type model interface {
	Name() string
	NewObj() interface{}
}

func Models(ms ...model) {
	for _, m := range ms {
		name := m.Name()
		if _, ok := _registry[name]; ok {
			panic(fmt.Sprintf("%s has already been registered", name))
		}
		_registry[name] = m
		_models = append(_models, m)
	}
}

func autoMigrate(ctx context.Context, db api.DB) error {
	var errs []error
	for _, m := range _models {
		if err := db.AutoMigrate(ctx, m.NewObj(), orm.WithTable(m.Name())); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.NewAggregate(errs)
}
