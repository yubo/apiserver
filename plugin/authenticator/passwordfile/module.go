package passwordfile

import (
	"context"

	"github.com/yubo/apiserver/components/dbus"
	"github.com/yubo/apiserver/pkg/authentication/authenticator"
	"github.com/yubo/apiserver/pkg/proc"
	authn "github.com/yubo/apiserver/pkg/server/authenticator"
	"github.com/yubo/apiserver/plugin/authenticator/basic"
	"k8s.io/klog/v2"
)

const (
	moduleName = "authentication.passwordAuthFile"
	configPath = "authentication"
)

func Register(opts ...proc.ModuleOption) {
	o := &proc.ModuleOptions{
		Proc: proc.DefaultProcess,
	}
	for _, v := range opts {
		v(o)
	}

	authn.RegisterAuthn(factory)
	o.Proc.AddConfig(configPath, newConfig(), proc.WithConfigGroup("authentication"))
}

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

	p, err := NewCSV(cf.PasswordAuthFile)
	if err != nil {
		return nil, err
	}

	dbus.RegisterPasswordfile(p)

	return basic.NewAuthenticator(p), nil
}
