package register

import (
	"github.com/yubo/apiserver/plugin/authenticator/token/bootstrap"
)

func init() {
	bootstrap.Register()
}
