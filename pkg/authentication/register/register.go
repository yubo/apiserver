package register

import (
	"github.com/yubo/apiserver/pkg/authentication"
)

func init() {
	authentication.Register()
}
