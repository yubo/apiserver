package session

import (
	"context"

	"github.com/yubo/apiserver/pkg/authentication"
	"github.com/yubo/apiserver/pkg/options"
	"github.com/yubo/apiserver/pkg/session"
	"github.com/yubo/golib/proc"
	"k8s.io/klog/v2"
)

const (
	moduleName       = "authentication"
	submoduleName    = "session"
	noUsernamePrefix = "-"
)

var (
	_auth   = &authModule{name: moduleName + ":" + submoduleName}
	hookOps = []proc.HookOps{{
		Hook:        _auth.init,
		Owner:       moduleName,
		HookNum:     proc.ACTION_START,
		Priority:    proc.PRI_SYS_INIT,
		SubPriority: options.PRI_M_AUTHN - 1,
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

func (p *authModule) init(ctx context.Context) error {
	c := proc.ConfigerFrom(ctx)

	cf := newConfig()
	if err := c.Read(moduleName, cf); err != nil {
		return err
	}
	p.config = cf

	if !cf.Session {
		klog.Infof("%s is disabled, skip", p.name)
		return nil
	}

	return authentication.RegisterAuthn(session.NewAuthenticator())
}

func init() {
	proc.RegisterHooks(hookOps)
	proc.RegisterFlags(moduleName, "authentication", newConfig())
}
