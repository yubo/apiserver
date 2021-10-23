package register

import (
	"context"

	"github.com/yubo/apiserver/pkg/authorization"
	"github.com/yubo/apiserver/pkg/authorization/authorizer"
	"github.com/yubo/apiserver/pkg/authorization/authorizerfactory"
)

const (
	moduleName = "authorization.AlwaysDeny"
)

func factory(_ context.Context) (authorizer.Authorizer, error) {
	return authorizerfactory.NewAlwaysDenyAuthorizer(), nil
}

func init() {
	authorization.RegisterAuthz("AlwaysDeny", factory)
}
