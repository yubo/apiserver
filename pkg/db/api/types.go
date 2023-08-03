package api

import "github.com/yubo/golib/orm"

type DB interface {
	orm.DB

	GetDB(name string) DB // panic if db[name] is not exist
}
