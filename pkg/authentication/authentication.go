package authentication

import (
	"context"
	"sort"
	"time"

	"github.com/yubo/apiserver/pkg/authentication/authenticator"
	"github.com/yubo/apiserver/pkg/authentication/group"
	"github.com/yubo/apiserver/pkg/authentication/request/anonymous"
	"github.com/yubo/apiserver/pkg/authentication/request/bearertoken"
	"github.com/yubo/apiserver/pkg/authentication/request/union"
	"github.com/yubo/apiserver/pkg/authentication/request/websocket"
	tokencache "github.com/yubo/apiserver/pkg/authentication/token/cache"
	"github.com/yubo/apiserver/pkg/authentication/token/tokenfile"
	tokenunion "github.com/yubo/apiserver/pkg/authentication/token/union"
	"github.com/yubo/apiserver/pkg/options"
	"github.com/yubo/golib/proc"
	"github.com/yubo/golib/staging/util/wait"
	"k8s.io/klog/v2"
)

const (
	moduleName = "authentication"
)

// config contains all authentication options for API Server
type config struct {
	//APIAudiences         []string      `json:"apiAudiences"`
	TokenSuccessCacheTTL time.Duration `json:"tokenSuccessCacheTTL" flag:"token-success-cache-ttl" default:"10s" description:"The duration to cache success token."`
	TokenFailureCacheTTL time.Duration `json:"tokenFailureCacheTTL" flag:"token-failure-cache-ttl" description:"The duration to cache failure token."`
	Anonymous            bool          `json:"anonymous" flag:"anonymous-auth" default:"true" description:"Enables anonymous requests to the secure port of the API server. Requests that are not rejected by another authentication method are treated as anonymous requests. Anonymous requests have a username of system:anonymous, and a group name of system:unauthenticated."`
}

// TokenFileAuthenticationOptions contains token file authentication options for API Server
type TokenFileAuthenticationOptions struct {
	TokenFile string
}

// WebHookAuthenticationOptions contains web hook authentication options for API Server
type WebHookAuthenticationOptions struct {
	ConfigFile string
	Version    string
	CacheTTL   time.Duration

	// RetryBackoff specifies the backoff parameters for the authentication webhook retry logic.
	// This allows us to configure the sleep time at each iteration and the maximum number of retries allowed
	// before we fail the webhook call in order to limit the fan out that ensues when the system is degraded.
	RetryBackoff *wait.Backoff
}

// newConfig create a new BuiltInAuthenticationOptions, just set default token cache TTL
func newConfig() *config {
	return &config{}
}

// Validate checks invalid config combination
func (o *config) Validate() error {
	return nil
}

type Authenticators []authenticator.Request

func (p Authenticators) Len() int {
	return len(p)
}

func (p Authenticators) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func (p Authenticators) Less(i, j int) bool {
	return p[i].Priority() < p[j].Priority()
}

type TokenAuthenticators []authenticator.Token

func (p TokenAuthenticators) Len() int {
	return len(p)
}

func (p TokenAuthenticators) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func (p TokenAuthenticators) Less(i, j int) bool {
	return p[i].Priority() < p[j].Priority()
}

func (p *authentication) initAuthentication() (err error) {
	c := p.config

	var authenticators []authenticator.Request
	var tokenAuthenticators TokenAuthenticators

	// token auth
	for _, v := range p.tokenAuthenticators {
		if !v.Available() {
			klog.V(5).Infof("authn.token.%s is invalid, skipping", v.Name())
			continue
		}
		tokenAuthenticators = append(tokenAuthenticators, v)
		klog.V(6).Infof("add %s tokenAuthenticator pri %d", v.Name(), v.Priority())
	}
	sort.Sort(tokenAuthenticators)

	// authn
	authns := make(Authenticators, len(p.authenticators))
	copy(authns, p.authenticators)

	if len(tokenAuthenticators) > 0 {
		tokenAuth := tokenunion.New(tokenAuthenticators...)
		if c.TokenSuccessCacheTTL > 0 || c.TokenFailureCacheTTL > 0 {
			tokenAuth = tokencache.New(tokenAuth, true,
				c.TokenSuccessCacheTTL, c.TokenFailureCacheTTL)
		}
		authns = append(authns,
			bearertoken.New(tokenAuth),
			websocket.NewProtocolAuthenticator(tokenAuth),
		)
	}
	sort.Sort(authns)

	for _, v := range authns {
		if !v.Available() {
			klog.V(5).Infof("authn.%s is invalid, skipping", v.Name())
			continue
		}
		authenticators = append(authenticators, v)
		klog.V(5).Infof("add %s tokenAuthenticator pri %d", v.Name(), v.Priority())
	}

	if len(authenticators) == 0 {
		if c.Anonymous {
			p.authenticator = anonymous.NewAuthenticator()
			klog.Infof("add anonymous authenticator")
			return nil
		}
		return nil
	}

	authenticator := union.New(authenticators...)
	authenticator = group.NewAuthenticatedGroupAdder(authenticator)

	if c.Anonymous {
		// If the authenticator chain returns an error, return an error (don't consider a bad bearer token
		// or invalid username/password combination anonymous).
		authenticator = union.NewFailOnError(authenticator, anonymous.NewAuthenticator())
		klog.Infof("add anonymous authenticator")
	}
	p.authenticator = authenticator
	return nil
}

// newAuthenticatorFromTokenFile returns an authenticator.Token or an error
func newAuthenticatorFromTokenFile(tokenAuthFile string) (authenticator.Token, error) {
	tokenAuthenticator, err := tokenfile.NewCSV(tokenAuthFile)
	if err != nil {
		return nil, err
	}

	return tokenAuthenticator, nil
}

var (
	_authn  = &authentication{name: moduleName}
	hookOps = []proc.HookOps{{
		Hook:        _authn.init,
		Owner:       moduleName,
		HookNum:     proc.ACTION_START,
		Priority:    proc.PRI_SYS_INIT,
		SubPriority: options.PRI_M_AUTHN,
	}, {
		Hook:        _authn.stop,
		Owner:       moduleName,
		HookNum:     proc.ACTION_STOP,
		Priority:    proc.PRI_SYS_START,
		SubPriority: options.PRI_M_AUTHN,
	}}
)

func RegisterAuthn(auth authenticator.Request) error {
	_authn.authenticators = append(_authn.authenticators, auth)
	return nil
}

func RegisterTokenAuthn(auth authenticator.Token) error {
	_authn.tokenAuthenticators = append(_authn.tokenAuthenticators, auth)
	return nil
}

type authentication struct {
	name                string
	config              *config
	authenticators      Authenticators      // registry
	tokenAuthenticators TokenAuthenticators // registry
	authenticator       authenticator.Request
	ctx                 context.Context
	cancel              context.CancelFunc
	stoppedCh           chan struct{}
}

func (p *authentication) Authenticator() authenticator.Request {
	return p.authenticator
}

func (p *authentication) init(ops *proc.HookOps) (err error) {
	ctx, c := ops.ContextAndConfiger()
	p.ctx, p.cancel = context.WithCancel(ctx)

	cf := &config{}
	if err := c.Read(moduleName, cf); err != nil {
		return err
	}
	p.config = cf

	if err = p.initAuthentication(); err != nil {
		return err
	}

	ops.SetContext(options.WithAuthn(ctx, p))
	return nil
}

func (p *authentication) stop(ops *proc.HookOps) error {
	if p.cancel == nil {
		return nil
	}

	p.cancel()

	//<-p.stoppedCh

	return nil
}

func Register() {
	proc.RegisterHooks(hookOps)

	proc.RegisterFlags(moduleName, "authentication", &config{})
}
