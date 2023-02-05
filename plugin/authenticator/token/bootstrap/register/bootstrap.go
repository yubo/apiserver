package register

import (
	"context"

	"github.com/yubo/apiserver/pkg/authentication"
	"github.com/yubo/apiserver/pkg/authentication/authenticator"
	"github.com/yubo/apiserver/pkg/models"
	"github.com/yubo/apiserver/plugin/authenticator/token/bootstrap"
	"github.com/yubo/apiserver/pkg/proc"
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

func newConfig() *config { return &config{BootstrapToken: true} }

func factory(ctx context.Context) (authenticator.Token, error) {
	cf := newConfig()
	if err := proc.ReadConfig(configPath, cf); err != nil {
		return nil, err
	}

	if !cf.BootstrapToken {
		klog.V(5).InfoS("skip authModule", "name", moduleName, "reason", "disabled")
		return nil, nil
	}

	klog.InfoS("authmodule init", "name", moduleName)

	return bootstrap.NewTokenAuthenticator(models.NewSecret()), nil
}

func init() {
	proc.AddConfig(configPath, newConfig(), proc.WithConfigGroup("authentication"))
	authentication.RegisterTokenAuthn(factory)
}
