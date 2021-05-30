package alwaysallow

import (
	"github.com/yubo/apiserver/pkg/authorization"
	"github.com/yubo/apiserver/pkg/authorization/authorizer"
	"github.com/yubo/apiserver/pkg/authorization/authorizerfactory"
)

const (
	moduleName    = "authorization"
	submoduleName = "AlwaysDeny"
)

func init() {
	factory := func() (authorizer.Authorizer, error) {
		return authorizerfactory.NewAlwaysDenyAuthorizer(), nil
	}
	if err := authorization.RegisterAuthz(submoduleName, factory); err != nil {
		panic(err)
	}
}
