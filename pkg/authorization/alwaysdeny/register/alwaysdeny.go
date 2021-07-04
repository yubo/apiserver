package register

import (
	"github.com/yubo/apiserver/pkg/authorization"
	"github.com/yubo/apiserver/pkg/authorization/authorizer"
	"github.com/yubo/apiserver/pkg/authorization/authorizerfactory"
)

const (
	moduleName = "authorization.AlwaysDeny"
)

func init() {
	factory := func() (authorizer.Authorizer, error) {
		return authorizerfactory.NewAlwaysDenyAuthorizer(), nil
	}
	if err := authorization.RegisterAuthz("AlwaysDeny", factory); err != nil {
		panic(err)
	}
}
