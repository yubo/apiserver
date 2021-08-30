package register

import (
	"context"

	"github.com/yubo/apiserver/pkg/authorization"
	"github.com/yubo/apiserver/pkg/authorization/abac"
	"github.com/yubo/apiserver/pkg/authorization/abac/api"
	"github.com/yubo/apiserver/pkg/authorization/authorizer"
	"github.com/yubo/apiserver/pkg/options"
	"github.com/yubo/golib/proc"
)

const (
	moduleName       = "authorization.ABAC"
	modulePath       = "authorization"
	noUsernamePrefix = "-"
)

var (
	_auth   = &authModule{name: moduleName}
	hookOps = []proc.HookOps{{
		Hook:        _auth.init,
		Owner:       moduleName,
		HookNum:     proc.ACTION_START,
		Priority:    proc.PRI_SYS_INIT,
		SubPriority: options.PRI_M_AUTHZ - 1,
	}}
	_config    *config
	PolicyList []*api.Policy
)

type config struct {
	PolicyFile string `json:"policyFile" flag:"authorization-policy-file" description:"File with authorization policy in json line by line format, used with --authorization-mode=ABAC, on the secure port."`
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

	return nil
}

func init() {
	proc.RegisterHooks(hookOps)
	proc.RegisterFlags(modulePath, "authorization", newConfig())

	factory := func() (authorizer.Authorizer, error) {
		if _auth.config.PolicyFile != "" {
			p, err := abac.NewFromFile(_auth.config.PolicyFile)
			if err != nil {
				return nil, err
			}
			return abac.PolicyList(append(PolicyList, p...)), nil
		}
		return abac.PolicyList(PolicyList), nil
	}

	authorization.RegisterAuthz("ABAC", factory)
}
