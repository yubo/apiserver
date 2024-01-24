package models

import (
	"github.com/yubo/apiserver/components/dbus"
	"github.com/yubo/golib/orm"
)

func DB() orm.DB {
	return dbus.DB()
}
