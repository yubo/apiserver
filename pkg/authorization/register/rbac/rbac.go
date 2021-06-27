package abac

import (
	"context"
	"fmt"

	"github.com/spf13/pflag"
	"github.com/yubo/apiserver/pkg/authorization"
	"github.com/yubo/apiserver/pkg/authorization/authorizer"
	"github.com/yubo/apiserver/pkg/authorization/rbac"
	"github.com/yubo/apiserver/pkg/listers"
	"github.com/yubo/apiserver/pkg/options"
	"github.com/yubo/golib/configer"
	"github.com/yubo/golib/proc"
	"github.com/yubo/golib/util"
)

const (
	moduleName       = "authorization"
	submoduleName    = "RBAC"
	noUsernamePrefix = "-"
)

var (
	_auth   = &authModule{name: moduleName + "." + submoduleName}
	hookOps = []proc.HookOps{{
		Hook:        _auth.init,
		Owner:       moduleName,
		HookNum:     proc.ACTION_START,
		Priority:    proc.PRI_SYS_INIT,
		SubPriority: options.PRI_M_AUTHZ - 1,
	}}
	_config *config
)

type config struct {
	//PolicyFile string `yaml:"policyFile"`
}

func (o *config) addFlags(fs *pflag.FlagSet) {
}

func (o *config) changed() interface{} {
	if o == nil {
		return nil
	}
	return util.Diff2Map(defaultConfig(), o)
}

func (o *config) Validate() error {
	return nil
}

type authModule struct {
	name   string
	ctx    context.Context
	config *config
}

func defaultConfig() *config {
	return &config{}
}

func (p *authModule) init(ops *proc.HookOps) error {
	ctx, c := ops.ContextAndConfiger()

	cf := defaultConfig()
	if err := c.ReadYaml(moduleName, cf,
		configer.WithOverride(_config.changed())); err != nil {
		return err
	}
	p.config = cf
	p.ctx = ctx

	return nil
}

func init() {
	proc.RegisterHooks(hookOps)
	_config = defaultConfig()
	_config.addFlags(proc.NamedFlagSets().FlagSet("authorization"))

	factory := func() (authorizer.Authorizer, error) {

		db, ok := options.DBFrom(_auth.ctx)
		if !ok {
			return nil, fmt.Errorf("unable to get db from the context")
		}
		return rbac.New(
			&rbac.RoleGetter{Lister: listers.NewRoleLister(db)},
			&rbac.RoleBindingLister{Lister: listers.NewRoleBindingLister(db)},
			&rbac.ClusterRoleGetter{Lister: listers.NewClusterRoleLister(db)},
			&rbac.ClusterRoleBindingLister{Lister: listers.NewClusterRoleBindingLister(db)},
		), nil
	}

	authorization.RegisterAuthz(submoduleName, factory)
}
