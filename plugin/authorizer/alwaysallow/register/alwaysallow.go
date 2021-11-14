package register

import (
	"context"

	"github.com/yubo/apiserver/pkg/authorization"
	"github.com/yubo/apiserver/pkg/authorization/authorizer"
	"github.com/yubo/apiserver/pkg/authorization/authorizerfactory"
)

const (
	modeName = "AlwaysAllow"
)

func factory(_ context.Context) (authorizer.Authorizer, error) {
	return authorizerfactory.NewAlwaysAllowAuthorizer(), nil
}

func init() {
	authorization.RegisterAuthz(modeName, factory)
}
