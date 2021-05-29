package oidc

import (
	"github.com/spf13/pflag"
	"github.com/yubo/apiserver/pkg/authentication/module"
	"github.com/yubo/apiserver/pkg/authentication/token/bootstrap"
	"github.com/yubo/apiserver/pkg/listers"
	"github.com/yubo/apiserver/pkg/options"
	"github.com/yubo/golib/proc"
	pconfig "github.com/yubo/golib/proc/config"
	"github.com/yubo/golib/util"
	"k8s.io/klog/v2"
)

const (
	moduleName       = "authentication"
	submoduleName    = "bootstrapToken"
	noUsernamePrefix = "-"
)

var (
	_auth   = &authModule{name: moduleName}
	hookOps = []proc.HookOps{{
		Hook:        _auth.init,
		Owner:       moduleName,
		HookNum:     proc.ACTION_START,
		Priority:    proc.PRI_SYS_INIT - 1,
		SubPriority: options.PRI_M_AUTHN,
	}}
	_config *config
)

type config struct {
	BootstrapToken bool `yaml:"bootstrapToken"`
}

func (o *config) addFlags(fs *pflag.FlagSet) {
	fs.BoolVar(&o.BootstrapToken, "enable-bootstrap-token-auth", o.BootstrapToken, ""+
		"Enable to allow secrets of type 'bootstrap.kubernetes.io/token' in the 'kube-system' "+
		"namespace to be used for TLS bootstrapping authentication.")
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
	return &config{BootstrapToken: true}
}

func (p *authModule) init(ops *proc.HookOps) error {
	ctx, configer := ops.ContextAndConfiger()

	cf := defaultConfig()
	if err := configer.ReadYaml(p.name, cf,
		pconfig.WithOverride(_config.changed())); err != nil {
		return err
	}
	p.config = cf

	if !cf.BootstrapToken {
		klog.Infof("%s is disabled, skip", p.name)
		return nil
	}

	return module.RegisterTokenAuthn(bootstrap.NewTokenAuthenticator(
		listers.NewSecretLister(options.DBMustFrom(ctx))))
}

func init() {
	proc.RegisterHooks(hookOps)
}
