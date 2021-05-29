package module

import (
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
	utilnet "github.com/yubo/golib/staging/util/net"
	"github.com/yubo/golib/staging/util/wait"
	"k8s.io/klog/v2"
)

// TODO: remvoe me
// Config contains the data on how to authenticate a request to the Kube API Server
type Authentication struct {
	Anonymous                   bool
	WebhookTokenAuthnConfigFile string
	WebhookTokenAuthnVersion    string
	WebhookTokenAuthnCacheTTL   time.Duration
	WebhookRetryBackoff         *wait.Backoff

	TokenSuccessCacheTTL time.Duration
	TokenFailureCacheTTL time.Duration

	// TODO, this is the only non-serializable part of the entire config.  Factor it out into a clientconfig
	//ServiceAccountTokenGetter   serviceaccount.ServiceAccountTokenGetter
	BootstrapTokenAuthenticator authenticator.Token

	// Optional field, custom dial function used to connect to webhook
	CustomDial utilnet.DialFunc

	// Authenticator determines which subject is making the request
	Authenticator authenticator.Request
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

func RegisterAuthn(auth authenticator.Request) error {
	_authn.authenticators = append(_authn.authenticators, auth)
	return nil
}

func RegisterTokenAuthn(auth authenticator.Token) error {
	_authn.tokenAuthenticators = append(_authn.tokenAuthenticators, auth)
	return nil
}

func (p *authentication) initAuthentication() (err error) {
	c := p.config
	authn := &Authentication{
		TokenSuccessCacheTTL: c.TokenSuccessCacheTTL,
		TokenFailureCacheTTL: c.TokenFailureCacheTTL,
		Anonymous:            c.Anonymous,
		//APIAudiences:         c.APIAudiences,
	}

	var authenticators []authenticator.Request
	var tokenAuthenticators []authenticator.Token

	// token auth
	sort.Sort(p.tokenAuthenticators)
	for _, v := range p.tokenAuthenticators {
		if !v.Available() {
			continue
		}
		tokenAuthenticators = append(tokenAuthenticators, v)
		klog.V(6).Infof("add %s tokenAuthenticator pri %d", v.Name(), v.Priority())
	}

	// authn
	authns := make(Authenticators, len(p.authenticators))
	copy(authns, p.authenticators)

	if len(tokenAuthenticators) > 0 {
		tokenAuth := tokenunion.New(tokenAuthenticators...)
		if authn.TokenSuccessCacheTTL > 0 || authn.TokenFailureCacheTTL > 0 {
			tokenAuth = tokencache.New(tokenAuth, true,
				authn.TokenSuccessCacheTTL, authn.TokenFailureCacheTTL)
		}
		authns = append(authns,
			bearertoken.New(tokenAuth),
			websocket.NewProtocolAuthenticator(tokenAuth),
		)
	}

	sort.Sort(authns)
	for _, v := range authns {
		if !v.Available() {
			continue
		}
		authenticators = append(authenticators, v)
		klog.V(6).Infof("add %s tokenAuthenticator pri %d", v.Name(), v.Priority())
	}

	if len(authenticators) == 0 {
		if authn.Anonymous {
			authn.Authenticator = anonymous.NewAuthenticator()
			klog.Infof("add anonymous authenticator")
			return nil
		}
		return nil
	}

	authenticator := union.New(authenticators...)
	authenticator = group.NewAuthenticatedGroupAdder(authenticator)

	if authn.Anonymous {
		// If the authenticator chain returns an error, return an error (don't consider a bad bearer token
		// or invalid username/password combination anonymous).
		authn.Authenticator = union.NewFailOnError(authenticator, anonymous.NewAuthenticator())
		klog.Infof("add anonymous authenticator")
	}
	p.authentication = authn
	return nil

	/*
		// ############################### old

		if c.WebHook != nil {
			authn.WebhookTokenAuthnConfigFile = c.WebHook.ConfigFile
			authn.WebhookTokenAuthnVersion = c.WebHook.Version
			authn.WebhookTokenAuthnCacheTTL = c.WebHook.CacheTTL
			authn.WebhookRetryBackoff = c.WebHook.RetryBackoff

			if len(c.WebHook.ConfigFile) > 0 && c.WebHook.CacheTTL > 0 {
				if c.TokenSuccessCacheTTL > 0 && c.WebHook.CacheTTL < c.TokenSuccessCacheTTL {
					klog.Warningf("the webhook cache ttl of %s is shorter than the overall cache ttl of %s for successful token authentication attempts.", c.WebHook.CacheTTL, c.TokenSuccessCacheTTL)
				}
				if c.TokenFailureCacheTTL > 0 && c.WebHook.CacheTTL < c.TokenFailureCacheTTL {
					klog.Warningf("the webhook cache ttl of %s is shorter than the overall cache ttl of %s for failed token authentication attempts.", c.WebHook.CacheTTL, c.TokenFailureCacheTTL)
				}
			}
		}

		if c.ServiceAccounts != nil && c.ServiceAccounts.Issuer != "" && len(c.APIAudiences) == 0 {
			authn.APIAudiences = authenticator.Audiences{c.ServiceAccounts.Issuer}
		}

	*/
}

// newAuthenticatorFromTokenFile returns an authenticator.Token or an error
func newAuthenticatorFromTokenFile(tokenAuthFile string) (authenticator.Token, error) {
	tokenAuthenticator, err := tokenfile.NewCSV(tokenAuthFile)
	if err != nil {
		return nil, err
	}

	return tokenAuthenticator, nil
}
