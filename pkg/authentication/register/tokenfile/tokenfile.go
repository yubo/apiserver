package oidc

import (
	"github.com/yubo/apiserver/pkg/authentication"
	"github.com/yubo/apiserver/pkg/authentication/token/tokenfile"
	"github.com/yubo/apiserver/pkg/options"
	"github.com/yubo/golib/proc"
	"k8s.io/klog/v2"
)

const (
	moduleName       = "authentication"
	submoduleName    = "tokenAuthFile"
	noUsernamePrefix = "-"
)

var (
	_auth   = &authModule{name: moduleName + "." + submoduleName}
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

func (p *authModule) init(ops *proc.HookOps) error {
	c := ops.Configer()

	cf := newConfig()
	if err := c.ReadYaml(moduleName, cf); err != nil {
		return err
	}
	p.config = cf

	if len(cf.TokenAuthFile) == 0 {
		klog.Infof("%s is not set, skip", p.name)
		return nil
	}

	auth, err := tokenfile.NewCSV(cf.TokenAuthFile)
	if err != nil {
		return err
	}

	return authentication.RegisterTokenAuthn(auth)
}

func init() {
	proc.RegisterHooks(hookOps)
	proc.RegisterFlags(moduleName, "authentication", newConfig())
}
