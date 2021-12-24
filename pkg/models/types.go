package models

import (
	"github.com/yubo/golib/queries"
)

type Model interface {
	List(selector queries.Selector, dst interface{}) error
	Get(name string, dst interface{}) error
	Create(obj interface{}) error
	Update(obj interface{}) error
	Delete(obj interface{}) error
}
