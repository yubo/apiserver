package register

import (
	"context"

	"github.com/yubo/apiserver/pkg/authentication"
	"github.com/yubo/apiserver/pkg/authentication/authenticator"
	"github.com/yubo/apiserver/pkg/models"
	"github.com/yubo/apiserver/plugin/authenticator/token/bootstrap"
	"github.com/yubo/golib/configer"
	"github.com/yubo/golib/proc"
	"k8s.io/klog/v2"
)

const (
	moduleName = "authentication.bootstrapToken"
	configPath = "authentication"
)

type config struct {
	BootstrapToken bool `json:"bootstrapToken" default:"true" flag:"enable-bootstrap-token-auth" description:"Enable to allow secrets of type 'bootstrap.kubernetes.io/token' in the 'kube-system' namespace to be used for TLS bootstrapping authentication."`
}

func (o *config) Validate() error {
	return nil
}

func newConfig() *config { return &config{} }

func factory(ctx context.Context) (authenticator.Token, error) {
	c := configer.ConfigerMustFrom(ctx)

	cf := newConfig()
	if err := c.Read(configPath, cf); err != nil {
		return nil, err
	}

	if !cf.BootstrapToken {
		klog.V(5).InfoS("skip authModule", "name", moduleName, "reason", "disabled")
		return nil, nil
	}

	klog.V(5).InfoS("authmodule init", "name", moduleName)

	return bootstrap.NewTokenAuthenticator(models.NewSecret()), nil
}

func init() {
	proc.RegisterFlags(configPath, "authentication", newConfig())
	authentication.RegisterTokenAuthn(factory)
}
