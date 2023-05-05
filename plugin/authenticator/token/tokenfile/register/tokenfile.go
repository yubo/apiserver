package register

import (
	"context"

	"github.com/yubo/apiserver/pkg/authentication"
	"github.com/yubo/apiserver/pkg/authentication/authenticator"
	"github.com/yubo/apiserver/pkg/proc"
	"github.com/yubo/apiserver/plugin/authenticator/token/tokenfile"
	"k8s.io/klog/v2"
)

const (
	moduleName = "authentication.tokenAuthFile"
	configPath = "authentication"
)

func newConfig() *config { return &config{} }

type config struct {
	TokenAuthFile string `json:"tokenAuthFile" flag:"token-auth-file" description:"If set, the file that will be used to secure the secure port of the API server via token authentication."`
}

func (o *config) Validate() error {
	return nil
}

func factory(ctx context.Context) (authenticator.Token, error) {
	cf := newConfig()
	if err := proc.ReadConfig(configPath, cf); err != nil {
		return nil, err
	}

	if len(cf.TokenAuthFile) == 0 {
		klog.V(5).InfoS("skip authModule", "name", moduleName, "reason", "tokenfile not set")
		return nil, nil
	}
	klog.V(5).InfoS("authmodule init", "name", moduleName, "file", cf.TokenAuthFile)

	return tokenfile.NewCSV(cf.TokenAuthFile)
}

func init() {
	authentication.RegisterTokenAuthn(factory)
	proc.AddConfig(configPath, newConfig(), proc.WithConfigGroup("authentication"))
}
