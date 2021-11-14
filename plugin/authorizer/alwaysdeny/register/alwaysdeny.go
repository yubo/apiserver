package register

import (
	"context"

	"github.com/yubo/apiserver/pkg/authorization"
	"github.com/yubo/apiserver/pkg/authorization/authorizer"
	"github.com/yubo/apiserver/pkg/authorization/authorizerfactory"
)

const (
	modeName = "AlwaysDeny"
)

func factory(_ context.Context) (authorizer.Authorizer, error) {
	return authorizerfactory.NewAlwaysDenyAuthorizer(), nil
}

func init() {
	authorization.RegisterAuthz(modeName, factory)
}
