package authenticator

import (
	"context"

	"github.com/yubo/apiserver/pkg/authentication/authenticator"
)

var (
	authenticatorFactories      []AuthenticatorFactory
	tokenAuthenticatorFactories []AuthenticatorTokenFactory
)

type AuthenticatorFactory func(context.Context) (authenticator.Request, error)
type AuthenticatorTokenFactory func(context.Context) (authenticator.Token, error)

func RegisterAuthn(factory AuthenticatorFactory) error {
	authenticatorFactories = append(authenticatorFactories, factory)
	return nil
}

func RegisterTokenAuthn(factory AuthenticatorTokenFactory) error {
	tokenAuthenticatorFactories = append(tokenAuthenticatorFactories, factory)
	return nil
}
