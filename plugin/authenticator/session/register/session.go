package session

import (
	"context"

	"github.com/yubo/apiserver/pkg/authentication"
	"github.com/yubo/apiserver/pkg/authentication/authenticator"
	"github.com/yubo/apiserver/pkg/proc/options"
	"github.com/yubo/apiserver/pkg/proc"
	"k8s.io/klog/v2"
)

const (
	moduleName       = "authentication.session"
	modulePath       = "authentication"
	noUsernamePrefix = "-"
)

var (
	_auth   = &authModule{name: moduleName}
	hookOps = []v1.HookOps{{
		Hook:        _auth.init,
		Owner:       moduleName,
		HookNum:     v1.ACTION_START,
		Priority:    v1.PRI_SYS_INIT,
		SubPriority: v1.PRI_M_AUTHN,
	}}
)

type config struct {
	Session bool `json:"session" default:"true" flag:"enable-session-auth" description:"Enable to allow session to be used for authentication."`
}

func (o *config) Validate() error {
	return nil
}

type authModule struct {
	name   string
	config *config
}

func newConfig() *config { return &config{} }

func factory(ctx context.Context) (authenticator.Request, error) {
	return NewAuthenticator(), nil
}

func (p *authModule) init(ctx context.Context) error {
	cf := newConfig()
	if err := proc.ReadConfig(modulePath, cf); err != nil {
		return err
	}
	p.config = cf

	if !cf.Session {
		klog.InfoS("skip authModule", "name", p.name, "reason", "disabled")
		return nil
	}
	klog.V(5).InfoS("authmodule init", "name", p.name)

	return authentication.RegisterAuthn(factory)
}

func init() {
	authentication.RegisterAuthn(factory)
	proc.RegisterHooks(hookOps)
	proc.AddConfig(modulePath, newConfig(), proc.WithConfigGroup("authentication"))
}
