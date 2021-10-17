package register

import (
	audit "github.com/yubo/apiserver/pkg/audit/module"
)

func init() {
	audit.Register()
}
