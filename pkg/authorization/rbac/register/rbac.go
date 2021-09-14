package register

import (
	"context"
	"fmt"

	"github.com/yubo/apiserver/pkg/authorization"
	"github.com/yubo/apiserver/pkg/authorization/authorizer"
	"github.com/yubo/apiserver/pkg/authorization/rbac"
	"github.com/yubo/apiserver/pkg/authorization/rbac/db"
	"github.com/yubo/apiserver/pkg/authorization/rbac/file"
	"github.com/yubo/apiserver/pkg/options"
	"github.com/yubo/golib/proc"
)

const (
	moduleName       = "authorization.rbac"
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
)

type config struct {
	file.Config
	Provider string `json:"provider" flag:"rbac-provider" description:"rbac provider(file,db), used with --authorization-mode=RBAC"`
}

func (o *config) Validate() error {
	return nil
}

type authModule struct {
	name   string
	ctx    context.Context
	config *config
}

func newConfig() *config {
	return &config{}
}

func (p *authModule) init(ctx context.Context) error {
	c := proc.ConfigerMustFrom(ctx)

	cf := newConfig()
	if err := c.Read(moduleName, cf); err != nil {
		return err
	}
	p.config = cf
	p.ctx = ctx

	var rbacAuth *rbac.RBACAuthorizer
	var err error
	switch cf.Provider {
	case "file":
		rbacAuth, err = file.NewRBAC(&cf.Config)
	case "db":
		rbacAuth, err = db.NewRBAC(options.DBMustFrom(ctx))
	case "":
	default:
		return fmt.Errorf("unsupported rbac provider %s", cf.Provider)
	}

	if err != nil {
		return err
	}
	options.WithRBAC(ctx, rbacAuth)

	return nil
}

func init() {
	proc.RegisterHooks(hookOps)
	proc.RegisterFlags(moduleName, "authorization", newConfig())

	factory := func() (authorizer.Authorizer, error) {
		rbac, ok := options.RBACFrom(_auth.ctx)
		if !ok {
			return nil, fmt.Errorf("unable to get db from the context")
		}

		return rbac, nil
	}
	authorization.RegisterAuthz("RBAC", factory)
}
