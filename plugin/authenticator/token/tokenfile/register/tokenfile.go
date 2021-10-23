package register

import (
	"context"

	"github.com/yubo/apiserver/pkg/authentication"
	"github.com/yubo/apiserver/plugin/authenticator/token/tokenfile"
	"github.com/yubo/apiserver/pkg/options"
	"github.com/yubo/golib/proc"
	"k8s.io/klog/v2"
)

const (
	moduleName = "authentication.tokenAuthFile"
	modulePath = "authentication"
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
	TokenAuthFile string `json:"tokenAuthFile" flag:"token-auth-file" description:"If set, the file that will be used to secure the secure port of the API server via token authentication."`
}

func (o *config) Validate() error {
	return nil
}

type authModule struct {
	name   string
	config *config
}

func newConfig() *config {
	return &config{}
}

func (p *authModule) init(ctx context.Context) error {
	c := proc.ConfigerMustFrom(ctx)

	cf := newConfig()
	if err := c.Read(modulePath, cf); err != nil {
		return err
	}
	p.config = cf

	if len(cf.TokenAuthFile) == 0 {
		klog.InfoS("skip authModule", "name", p.name, "reason", "tokenfile not set")
		return nil
	}
	klog.V(5).InfoS("authmodule init", "name", p.name, "file", cf.TokenAuthFile)

	auth, err := tokenfile.NewCSV(cf.TokenAuthFile)
	if err != nil {
		return err
	}

	return authentication.RegisterTokenAuthn(auth)
}

func init() {
	proc.RegisterHooks(hookOps)
	proc.RegisterFlags(modulePath, "authentication", newConfig())
}
