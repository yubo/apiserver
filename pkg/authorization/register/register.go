package register

import (
	"github.com/yubo/apiserver/pkg/authorization"
)

func init() {
	authorization.Register()
}
