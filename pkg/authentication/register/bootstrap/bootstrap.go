package oidc

import (
	"fmt"

	"github.com/spf13/pflag"
	"github.com/yubo/apiserver/pkg/authentication"
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
		Priority:    proc.PRI_SYS_INIT,
		SubPriority: options.PRI_M_AUTHN - 1,
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

	db, ok := options.DBFrom(ctx)
	if !ok {
		return fmt.Errorf("unable to get db from the context")
	}

	return authentication.RegisterTokenAuthn(bootstrap.NewTokenAuthenticator(
		listers.NewSecretLister(db)))
}

func init() {
	proc.RegisterHooks(hookOps)
	_config = defaultConfig()
	_config.addFlags(proc.NamedFlagSets().FlagSet("authentication"))
}
