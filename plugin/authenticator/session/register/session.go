package session

import (
	"context"

	"github.com/yubo/apiserver/pkg/authentication/authenticator"
	"github.com/yubo/apiserver/pkg/proc"
	authn "github.com/yubo/apiserver/pkg/server/authenticator"
	"github.com/yubo/apiserver/plugin/authenticator/session"
	"k8s.io/klog/v2"
)

const (
	moduleName = "authentication"
)

func newConfig() *config { return &config{} }

type config struct {
	Session bool `json:"session" flag:"enable-session-auth" description:"Enable to allow session to be used for authentication."`
}

func (o *config) Validate() error {
	return nil
}

func factory(ctx context.Context) (authenticator.Request, error) {
	cf := newConfig()
	if err := proc.ReadConfig(moduleName, cf); err != nil {
		return nil, err
	}

	if !cf.Session {
		klog.InfoS("skip authModule", "name", moduleName, "reason", "disabled")
		return nil, nil
	}

	klog.V(5).InfoS("authmodule init", "name", moduleName)
	return session.NewAuthenticator(), nil
}

func init() {
	authn.RegisterAuthn(factory)
	proc.AddConfig(moduleName, newConfig(), proc.WithConfigGroup("authentication"))
}
