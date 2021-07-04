package register

import (
	"github.com/yubo/apiserver/pkg/authorization"
	"github.com/yubo/apiserver/pkg/authorization/authorizer"
	"github.com/yubo/apiserver/pkg/authorization/authorizerfactory"
)

const (
	moduleName = "authorization.AlwaysAllow"
)

func init() {
	factory := func() (authorizer.Authorizer, error) {
		return authorizerfactory.NewAlwaysAllowAuthorizer(), nil
	}
	if err := authorization.RegisterAuthz("AlwaysAllow", factory); err != nil {
		panic(err)
	}
}
