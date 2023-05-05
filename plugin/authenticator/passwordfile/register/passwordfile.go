package passwordfile

import (
	"context"

	"github.com/yubo/apiserver/pkg/authentication"
	"github.com/yubo/apiserver/pkg/authentication/authenticator"
	"github.com/yubo/apiserver/pkg/proc"
	"github.com/yubo/apiserver/plugin/authenticator/basic"
	"github.com/yubo/apiserver/plugin/authenticator/passwordfile"
	"k8s.io/klog/v2"
)

const (
	moduleName = "authentication.passwordAuthFile"
	configPath = "authentication"
)

func newConfig() *config {
	return &config{}
}

type config struct {
	PasswordAuthFile string `json:"passwordAuthFile" flag:"password-auth-file" description:"If set, the file that will be used to secure the secure port of the API server via password authentication."`
}

func (o *config) Validate() error {
	return nil
}

func factory(ctx context.Context) (authenticator.Request, error) {
	cf := newConfig()
	if err := proc.ReadConfig(configPath, cf); err != nil {
		return nil, err
	}
	if cf.PasswordAuthFile == "" {
		klog.InfoS("skip authModule", "name", moduleName, "reason", "noset")
		return nil, nil
	}

	p, err := passwordfile.NewCSV(cf.PasswordAuthFile)
	if err != nil {
		return nil, err
	}

	return basic.NewAuthenticator(p), nil
}

func init() {
	authentication.RegisterAuthn(factory)
	proc.AddConfig(configPath, newConfig(), proc.WithConfigGroup("authentication"))
}
