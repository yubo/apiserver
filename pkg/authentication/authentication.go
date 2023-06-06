package authentication

import (
	"context"

	"github.com/yubo/apiserver/components/dbus"
	"github.com/yubo/apiserver/pkg/authentication/authenticator"
	"github.com/yubo/apiserver/pkg/authentication/group"
	"github.com/yubo/apiserver/pkg/authentication/request/anonymous"
	"github.com/yubo/apiserver/pkg/authentication/request/bearertoken"
	"github.com/yubo/apiserver/pkg/authentication/request/union"
	"github.com/yubo/apiserver/pkg/authentication/request/websocket"
	tokencache "github.com/yubo/apiserver/pkg/authentication/token/cache"
	tokenunion "github.com/yubo/apiserver/pkg/authentication/token/union"
	"github.com/yubo/apiserver/pkg/proc"
	v1 "github.com/yubo/apiserver/pkg/proc/api/v1"
	"github.com/yubo/apiserver/pkg/server"
	"github.com/yubo/golib/api"
	"github.com/yubo/golib/util"
)

const (
	moduleName = "authentication"
)

type key int

const (
	configKey key = iota
)

func WithConfig(parent context.Context, config *Config) context.Context {
	return context.WithValue(parent, configKey, config)
}

func ConfigFrom(ctx context.Context) *Config {
	config, ok := ctx.Value(configKey).(*Config)
	if !ok {
		panic("unable to get authn.config from context")
	}
	return config
}

// Config contains all authentication options for API Server
type Config struct {
	APIAudiences         []string     `json:"apiAudiences" flag:"api-audiences" description:"Identifiers of the API. The service account token authenticator will validate that tokens used against the API are bound to at least one of these audiences. If the --service-account-issuer flag is configured and this flag is not, this field defaults to a single element list containing the issuer URL."`
	TokenSuccessCacheTTL api.Duration `json:"tokenSuccessCacheTTL" flag:"token-success-cache-ttl" default:"10s" description:"The duration to cache success token."`
	TokenFailureCacheTTL api.Duration `json:"tokenFailureCacheTTL" flag:"token-failure-cache-ttl" description:"The duration to cache failure token."`
	Anonymous            bool         `json:"anonymous" flag:"anonymous-auth" default:"false" description:"Enables anonymous requests to the secure port of the API server. Requests that are not rejected by another authentication method are treated as anonymous requests. Anonymous requests have a username of system:anonymous, and a group name of system:unauthenticated."`
}

// newConfig create a new BuiltInAuthenticationOptions, just set default token cache TTL
func newConfig() *Config {
	return &Config{}
}

// Validate checks invalid config combination
func (p *Config) Validate() error {
	return nil
}

func (p *authentication) initAuthentication() (err error) {
	var authenticators []authenticator.Request
	var tokenAuthenticators []authenticator.Token

	config := p.config

	p.ctx = WithConfig(p.ctx, config)

	for _, factory := range p.authenticatorFactories {
		auth, err := factory(p.ctx)
		if err != nil {
			return err
		}
		if !util.IsNil(auth) {
			authenticators = append(authenticators, auth)
		}
	}

	for _, factory := range p.tokenAuthenticatorFactories {
		token, err := factory(p.ctx)
		if err != nil {
			return err
		}
		if !util.IsNil(token) {
			tokenAuthenticators = append(tokenAuthenticators, token)
		}
	}

	if len(tokenAuthenticators) > 0 {
		tokenAuth := tokenunion.New(tokenAuthenticators...)
		if config.TokenSuccessCacheTTL.Duration > 0 || config.TokenFailureCacheTTL.Duration > 0 {
			tokenAuth = tokencache.New(tokenAuth, true,
				config.TokenSuccessCacheTTL.Duration, config.TokenFailureCacheTTL.Duration)
		}
		authenticators = append(authenticators,
			bearertoken.New(tokenAuth),
			websocket.NewProtocolAuthenticator(tokenAuth),
		)
	}

	if len(authenticators) == 0 {
		if config.Anonymous {
			p.authenticator = anonymous.NewAuthenticator()
			return nil
		}
		return nil
	}

	authenticator := union.New(authenticators...)
	authenticator = group.NewAuthenticatedGroupAdder(authenticator)

	if config.Anonymous {
		// If the authenticator chain returns an error, return an error (don't consider a bad bearer token
		// or invalid username/password combination anonymous).
		authenticator = union.NewFailOnError(authenticator, anonymous.NewAuthenticator())
	}
	p.authenticator = authenticator
	return nil
}

var (
	_authn  = &authentication{name: moduleName}
	hookOps = []v1.HookOps{{
		Hook:        _authn.init,
		Owner:       moduleName,
		HookNum:     v1.ACTION_START,
		Priority:    v1.PRI_SYS_INIT,
		SubPriority: v1.PRI_M_AUTHN,
	}, {
		Hook:        _authn.stop,
		Owner:       moduleName,
		HookNum:     v1.ACTION_STOP,
		Priority:    v1.PRI_SYS_START,
		SubPriority: v1.PRI_M_AUTHN,
	}}
)

type AuthenticatorFactory func(context.Context) (authenticator.Request, error)
type AuthenticatorTokenFactory func(context.Context) (authenticator.Token, error)

func RegisterAuthn(factory AuthenticatorFactory) error {
	_authn.authenticatorFactories = append(_authn.authenticatorFactories, factory)
	return nil
}

func RegisterTokenAuthn(factory AuthenticatorTokenFactory) error {
	_authn.tokenAuthenticatorFactories = append(_authn.tokenAuthenticatorFactories, factory)
	return nil
}

func APIAudiences() authenticator.Audiences {
	return authenticator.Audiences(_authn.config.APIAudiences)
}

type authentication struct {
	name                        string
	config                      *Config
	authenticatorFactories      []AuthenticatorFactory
	tokenAuthenticatorFactories []AuthenticatorTokenFactory
	authenticator               authenticator.Request
	ctx                         context.Context
	cancel                      context.CancelFunc
	stoppedCh                   chan struct{}
}

func (p *authentication) init(ctx context.Context) error {
	p.ctx, p.cancel = context.WithCancel(ctx)

	cf := newConfig()
	if err := proc.ReadConfig(moduleName, cf); err != nil {
		return err
	}
	p.config = cf

	if err := p.initAuthentication(); err != nil {
		return err
	}

	authn := &server.AuthenticationInfo{
		APIAudiences:  authenticator.Audiences(p.config.APIAudiences),
		Authenticator: p.authenticator,
		Anonymous:     p.config.Anonymous,
	}

	dbus.RegisterAuthenticationInfo(authn)
	return nil
}

func (p *authentication) stop(ctx context.Context) error {
	if p.cancel != nil {
		p.cancel()
	}
	return nil
}

func Register() {
	proc.RegisterHooks(hookOps)
	proc.AddConfig(moduleName, newConfig(), proc.WithConfigGroup("authentication"))
}
