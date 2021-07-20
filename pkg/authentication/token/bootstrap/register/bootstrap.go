package register

import (
	"context"
	"fmt"

	"github.com/yubo/apiserver/pkg/authentication"
	"github.com/yubo/apiserver/pkg/authentication/token/bootstrap"
	"github.com/yubo/apiserver/pkg/listers"
	"github.com/yubo/apiserver/pkg/options"
	"github.com/yubo/golib/proc"
	"k8s.io/klog/v2"
)

const (
	moduleName       = "authentication.bootstrapToken"
	modulePath       = "authentication"
	noUsernamePrefix = "-"
)

var (
	_auth   = &authModule{name: moduleName}
	hookOps = []proc.HookOps{{
		Hook:        _auth.init,
		Owner:       moduleName,
		HookNum:     proc.ACTION_START,
		Priority:    proc.PRI_SYS_INIT,
		SubPriority: options.PRI_M_AUTHN - 1,
	}}
)

type config struct {
	BootstrapToken bool `json:"bootstrapToken" default:"true" flag:"enable-bootstrap-token-auth" description:"Enable to allow secrets of type 'bootstrap.kubernetes.io/token' in the 'kube-system' namespace to be used for TLS bootstrapping authentication."`
}

func (o *config) Validate() error {
	return nil
}

type authModule struct {
	name   string
	config *config
}

func newConfig() *config { return &config{} }

func (p *authModule) init(ctx context.Context) error {
	c := proc.ConfigerFrom(ctx)

	cf := newConfig()
	if err := c.Read(modulePath, cf); err != nil {
		return err
	}
	p.config = cf

	if !cf.BootstrapToken {
		klog.InfoS("skip authModule", "name", p.name, "reason", "disabled")
		return nil
	}

	db, ok := options.DBFrom(ctx)
	if !ok {
		return fmt.Errorf("unable to get db from the context")
	}
	klog.V(5).InfoS("authmodule init", "name", p.name)

	return authentication.RegisterTokenAuthn(bootstrap.NewTokenAuthenticator(
		listers.NewSecretLister(db)))
}

func init() {
	proc.RegisterHooks(hookOps)
	proc.RegisterFlags(modulePath, "authentication", newConfig())
}