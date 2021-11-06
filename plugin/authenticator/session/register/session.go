package session

import (
	"context"

	"github.com/yubo/apiserver/pkg/authentication"
	"github.com/yubo/apiserver/pkg/authentication/authenticator"
	"github.com/yubo/apiserver/pkg/options"
	"github.com/yubo/apiserver/pkg/session"
	"github.com/yubo/golib/proc"
	"k8s.io/klog/v2"
)

const (
	moduleName       = "authentication.session"
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
		SubPriority: options.PRI_M_AUTHN,
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
	return session.NewAuthenticator(), nil
}

func (p *authModule) init(ctx context.Context) error {
	c := proc.ConfigerMustFrom(ctx)

	cf := newConfig()
	if err := c.Read(modulePath, cf); err != nil {
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
	proc.RegisterHooks(hookOps)
	proc.RegisterFlags(modulePath, "authentication", newConfig())
}
