package register

import (
	db "github.com/yubo/apiserver/pkg/db/module"
)

func init() {
	db.Register()
}
