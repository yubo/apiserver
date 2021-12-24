package models

import (
	"errors"
	"fmt"

	"github.com/yubo/golib/orm"
	"github.com/yubo/golib/queries"
)

var (
	catalog        = map[string]Model{}
	db             orm.DB
	ErrUnsupported = errors.New("Unsupported")
)

type baseModel struct {
	kind string
}

func (p baseModel) list(selector queries.Selector, dst interface{}) error {
	return nil
}
func (p baseModel) get(name string, dst interface{}) error {
	return nil
}
func (p baseModel) create(obj interface{}) error { return nil }
func (p baseModel) update(obj interface{}) error { return nil }
func (p baseModel) delete(obj interface{}) error { return nil }

// RegisterModel adds a model to the catalog
func RegisterModel(name string, m Model) error {
	_, found := catalog[name]
	if found {
		return fmt.Errorf("model %s has been registered", name)
	}
	catalog[name] = m
	return nil
}
