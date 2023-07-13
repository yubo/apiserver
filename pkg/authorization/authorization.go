package authorization

import (
	"context"
	"fmt"
	"runtime/debug"
	"strings"

	"github.com/yubo/apiserver/components/dbus"
	"github.com/yubo/apiserver/pkg/authorization/authorizer"
	"github.com/yubo/apiserver/pkg/authorization/authorizerfactory"
	"github.com/yubo/apiserver/pkg/authorization/path"
	"github.com/yubo/apiserver/pkg/authorization/union"
	"github.com/yubo/apiserver/pkg/proc"
	v1 "github.com/yubo/apiserver/pkg/proc/api/v1"
	"github.com/yubo/apiserver/pkg/server"
	"github.com/yubo/golib/configer"
	"github.com/yubo/golib/util"
	utilerrors "github.com/yubo/golib/util/errors"
	"github.com/yubo/golib/util/sets"
	"k8s.io/klog/v2"
)

const (
	moduleName = "authorization"
)

var (
	_authz = &authorization{
		name:                moduleName,
		authorizerFactories: map[string]authorizer.AuthorizerFactory{},
	}
	hookOps = []v1.HookOps{{
		Hook:        _authz.init,
		Owner:       moduleName,
		HookNum:     v1.ACTION_START,
		Priority:    v1.PRI_SYS_INIT,
		SubPriority: v1.PRI_M_AUTHZ,
	}, {
		Hook:        _authz.stop,
		Owner:       moduleName,
		HookNum:     v1.ACTION_STOP,
		Priority:    v1.PRI_SYS_START,
		SubPriority: v1.PRI_M_AUTHZ,
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
	Modes []string `json:"modes" flag:"authorization-mode" description:"Ordered list of plug-ins to do authorization on secure port."`

	// AlwaysAllowPaths are HTTP paths which are excluded from authorization. They can be plain
	// paths or end in * in which case prefix-match is applied. A leading / is optional.
	AlwaysAllowPaths []string `json:"alwaysAllowPaths" flag:"authorization-always-allow-paths" description:"A list of HTTP paths to skip during authorization, i.e. these are authorized without contacting the 'core' kubernetes server." default:"/healthz /readyz /livez"`

	// AlwaysAllowGroups are groups which are allowed to take any actions.  In kube, this is system:masters.
	AlwaysAllowGroups []string `json:"alwaysAllowGroups" flag:"authorization-always-allow-groups" description:"AlwaysAllowGroups are groups which are allowed to take any actions." default:"system:masters"`
}

func (p *config) GetTags() map[string]*configer.FieldTag {
	defaultMode := "AlwaysAllow"
	if len(AuthorizationModeChoices) > 0 && !IsValidAuthorizationMode("AlwaysAllow") {
		defaultMode = AuthorizationModeChoices[0]
	}
	return map[string]*configer.FieldTag{
		"modes": {
			Description: "Ordered list of plug-ins to do authorization on secure port. Comma-delimited list of: " + strings.Join(AuthorizationModeChoices, ",") + ".",
			Default:     defaultMode,
		},
	}
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
			debug.PrintStack()
			allErrors = append(allErrors,
				fmt.Errorf("authorization-mode %q is not a valid mode, modes %+v", mode, AuthorizationModeChoices))
		}
	}

	if len(o.Modes) != len(modes.List()) {
		allErrors = append(allErrors, fmt.Errorf("authorization-mode %q has mode specified more than once", o.Modes))
	}

	return utilerrors.NewAggregate(allErrors)
}

type authorization struct {
	name   string
	config *config

	authorizer          authorizer.Authorizer
	authorizerFactories map[string]authorizer.AuthorizerFactory

	ctx       context.Context
	cancel    context.CancelFunc
	stoppedCh chan struct{}
}

func RegisterAuthz(name string, factory authorizer.AuthorizerFactory) error {
	if _, ok := _authz.authorizerFactories[name]; ok {
		//return fmt.Errorf("authz %q is already registered", name)
		panic(fmt.Sprintf("authz %q is already registered", name))
	}
	_authz.authorizerFactories[name] = factory

	AuthorizationModeChoices = append(AuthorizationModeChoices, name)
	return nil
}

func (p *authorization) init(ctx context.Context) error {
	p.ctx, p.cancel = context.WithCancel(ctx)

	cf := newConfig()
	if err := proc.ReadConfig(p.name, cf); err != nil {
		return err
	}
	p.config = cf

	if err := p.initAuthorization(); err != nil {
		return err
	}

	authz := &server.AuthorizationInfo{
		Authorizer: p.authorizer,
		Modes:      sets.NewString(cf.Modes...),
	}

	klog.InfoS("withAuthz", "modes", cf.Modes)
	dbus.RegisterAuthorizationInfo(authz)

	return nil
}

func (p *authorization) stop(ctx context.Context) error {
	p.cancel()

	return nil
}

func (p *authorization) initAuthorization() (err error) {
	c := p.config

	klog.V(5).Infof("authz %+v", c.Modes)
	if len(c.Modes) == 0 {
		return fmt.Errorf("at least one authz mode must be passed")
	}

	if klog.V(6).Enabled() {
		for k := range p.authorizerFactories {
			klog.Infof("authz %s is valid", k)
		}
	}

	var authorizers []authorizer.Authorizer

	for _, mode := range c.Modes {
		factory, ok := p.authorizerFactories[mode]
		if !ok {
			return fmt.Errorf("unknown authz.%s specified", mode)
		}

		if factory == nil {
			klog.V(5).Infof("authz.%s is nil, skip", mode)
			continue
		}

		authz, err := factory(p.ctx)
		if err != nil {
			return fmt.Errorf("authz.%s error %s", mode, err)
		}
		authorizers = append(authorizers, authz)
		klog.V(5).Infof("authz.%s loaded", mode)
	}

	if len(c.AlwaysAllowGroups) > 0 {
		authorizers = append(authorizers, authorizerfactory.NewPrivilegedGroups(c.AlwaysAllowGroups...))
	}

	if len(c.AlwaysAllowPaths) > 0 {
		a, err := path.NewAuthorizer(c.AlwaysAllowPaths)
		if err != nil {
			return err
		}
		authorizers = append(authorizers, a)
	}

	p.authorizer = union.New(authorizers...)

	return nil
}
func RegisterHooks() {
	proc.RegisterHooks(hookOps)
}

func RegisterFlags() {
	cf := newConfig()
	proc.AddConfig(moduleName, cf, proc.WithConfigGroup("authorization"))
}

func Register() {
	RegisterHooks()
	RegisterFlags()
}
