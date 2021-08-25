package authorization

import (
	"context"
	"fmt"

	"github.com/yubo/apiserver/pkg/authorization/authorizer"
	"github.com/yubo/apiserver/pkg/authorization/union"
	"github.com/yubo/apiserver/pkg/options"
	"github.com/yubo/golib/proc"
	"github.com/yubo/golib/util"
	utilerrors "github.com/yubo/golib/util/errors"
	"github.com/yubo/golib/util/sets"
	"k8s.io/klog/v2"
)

const (
	// ModeAlwaysAllow is the mode to set all requests as authorized
	ModeAlwaysAllow string = "AlwaysAllow"
)

const (
	moduleName = "authorization"
)

var (
	_authz = &authorization{
		name:          moduleName,
		authzFactorys: map[string]authorizer.AuthorizerFactory{},
	}
	hookOps = []proc.HookOps{{
		Hook:        _authz.init,
		Owner:       moduleName,
		HookNum:     proc.ACTION_START,
		Priority:    proc.PRI_SYS_INIT,
		SubPriority: options.PRI_M_AUTHZ,
	}, {
		Hook:        _authz.stop,
		Owner:       moduleName,
		HookNum:     proc.ACTION_STOP,
		Priority:    proc.PRI_SYS_START,
		SubPriority: options.PRI_M_AUTHZ,
	}}
	//Config *config

	// AuthorizationModeChoices is the list of supported authorization modes
	AuthorizationModeChoices = []string{}
)

func IsValidAuthorizationMode(authzMode string) bool {
	return sets.NewString(AuthorizationModeChoices...).Has(authzMode)
}

// config contains all build-in authorization options for API Server
type config struct {
	Modes []string `json:"modes" default:"AlwaysAllow" flag:"authorization-mode" description:"Ordered list of plug-ins to do authorization on secure port."`
}

// newConfig create a config with default value
func newConfig() *config {
	return &config{ //Modes: []string{ModeAlwaysAllow},
	}
}
func (o *config) String() string {
	return util.Prettify(o)
}

// Validate checks invalid config combination
func (o *config) Validate() error {
	if o == nil {
		return nil
	}
	allErrors := []error{}

	if len(o.Modes) == 0 {
		allErrors = append(allErrors, fmt.Errorf("at least one authorization-mode must be passed"))
	}

	modes := sets.NewString(o.Modes...)
	for _, mode := range o.Modes {
		if !IsValidAuthorizationMode(mode) {
			allErrors = append(allErrors,
				fmt.Errorf("authorization-mode %q is not a valid mode, modes %+v", mode, AuthorizationModeChoices))
		}
	}

	if len(o.Modes) != len(modes.List()) {
		allErrors = append(allErrors, fmt.Errorf("authorization-mode %q has mode specified more than once", o.Modes))
	}

	return utilerrors.NewAggregate(allErrors)
}

// addFlags returns flags of authorization for a API Server
//func (o *config) addFlags(fs *pflag.FlagSet) {
//	fs.StringSliceVar(&o.Modes, "authorization-mode", o.Modes, ""+
//		"Ordered list of plug-ins to do authorization on secure port. Comma-delimited list of: "+
//		strings.Join(AuthorizationModeChoices, ",")+".")
//}

type authorization struct {
	name   string
	config *config
	//authorization *Authorization

	authorizer    authorizer.Authorizer
	authzFactorys map[string]authorizer.AuthorizerFactory

	ctx       context.Context
	cancel    context.CancelFunc
	stoppedCh chan struct{}
}

func RegisterAuthz(name string, factory authorizer.AuthorizerFactory) error {
	if _, ok := _authz.authzFactorys[name]; ok {
		return fmt.Errorf("authz %q is already registered", name)
	}
	_authz.authzFactorys[name] = factory

	AuthorizationModeChoices = append(AuthorizationModeChoices, name)
	return nil
}

func (p *authorization) Authorizer() authorizer.Authorizer {
	return p.authorizer
}

func (p *authorization) init(ctx context.Context) error {
	c := proc.ConfigerMustFrom(ctx)
	p.ctx, p.cancel = context.WithCancel(ctx)

	cf := newConfig()
	if err := c.Read(p.name, cf); err != nil {
		return err
	}
	p.config = cf

	if err := p.initAuthorization(); err != nil {
		return err
	}

	options.WithAuthz(ctx, p)

	return nil
}

func (p *authorization) stop(ctx context.Context) error {
	p.cancel()

	return nil
}

func RegisterHooks() {
	proc.RegisterHooks(hookOps)
}

func RegisterFlags() {
	proc.RegisterFlags(moduleName, moduleName, newConfig())
}

func Register() {
	RegisterHooks()
	RegisterFlags()
}

func (p *authorization) initAuthorization() (err error) {
	c := p.config

	klog.V(5).Infof("authz %+v", c.Modes)
	if len(c.Modes) == 0 {
		return fmt.Errorf("at least one authz mode must be passed")
	}

	if klog.V(6).Enabled() {
		for k, _ := range p.authzFactorys {
			klog.Infof("authz %s is valid", k)
		}
	}

	var authorizers []authorizer.Authorizer

	for _, mode := range c.Modes {
		factory, ok := p.authzFactorys[mode]
		if !ok {
			return fmt.Errorf("unknown authz mode %s specified", mode)
		}

		if factory == nil {
			klog.V(5).Infof("authz factory %q is nil, skip", mode)
			continue
		}

		authz, err := factory()
		if err != nil {
			return err
		}
		authorizers = append(authorizers, authz)
		klog.V(5).Infof("authz %s loaded", mode)
	}
	p.authorizer = union.New(authorizers...)

	return nil
}
