package abac

import (
	"github.com/spf13/pflag"
	"github.com/yubo/apiserver/pkg/authorization"
	"github.com/yubo/apiserver/pkg/authorization/abac"
	"github.com/yubo/apiserver/pkg/authorization/authorizer"
	"github.com/yubo/apiserver/pkg/options"
	"github.com/yubo/golib/proc"
	pconfig "github.com/yubo/golib/proc/config"
	"github.com/yubo/golib/util"
)

const (
	moduleName       = "authorization"
	submoduleName    = "ABAC"
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
	PolicyFile string `yaml:"policyFile"`
}

func (o *config) addFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.PolicyFile, "authorization-policy-file", o.PolicyFile, ""+
		"File with authorization policy in json line by line format, used with --authorization-mode=ABAC, on the secure port.")
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
	config *config
}

func defaultConfig() *config {
	return &config{}
}

func (p *authModule) init(ops *proc.HookOps) error {
	configer := ops.Configer()

	cf := defaultConfig()
	if err := configer.ReadYaml(moduleName, cf,
		pconfig.WithOverride(_config.changed())); err != nil {
		return err
	}
	p.config = cf

	return nil
}

func init() {
	proc.RegisterHooks(hookOps)
	_config = defaultConfig()
	_config.addFlags(proc.NamedFlagSets().FlagSet("authorization"))

	factory := func() (authorizer.Authorizer, error) {
		return abac.NewFromFile(_auth.config.PolicyFile)
	}

	authorization.RegisterAuthz(submoduleName, factory)

}
