package register

import (
	session "github.com/yubo/apiserver/pkg/session/module"
)

func init() {
	session.Register()
}
