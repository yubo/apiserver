package oidc

import (
	"github.com/spf13/pflag"
	"github.com/yubo/apiserver/pkg/authentication"
	"github.com/yubo/apiserver/pkg/authentication/token/tokenfile"
	"github.com/yubo/apiserver/pkg/options"
	"github.com/yubo/golib/proc"
	pconfig "github.com/yubo/golib/proc/config"
	"github.com/yubo/golib/util"
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
	_config *config
)

type config struct {
	TokenAuthFile string `yaml:"tokenAuthFile"`
}

func (o *config) addFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.TokenAuthFile, "token-auth-file", o.TokenAuthFile, ""+
		"If set, the file that will be used to secure the secure port of the API server "+
		"via token authentication.")

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
	_config = defaultConfig()
	_config.addFlags(proc.NamedFlagSets().FlagSet("authentication"))
}
