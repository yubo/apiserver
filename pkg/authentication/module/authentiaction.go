package authentication

import (
	"context"
	"time"

	"github.com/yubo/apiserver/pkg/authentication/authenticator"
	"github.com/yubo/apiserver/pkg/authentication/group"
	"github.com/yubo/apiserver/pkg/authentication/request/anonymous"
	"github.com/yubo/apiserver/pkg/authentication/request/bearertoken"
	"github.com/yubo/apiserver/pkg/authentication/request/union"
	"github.com/yubo/apiserver/pkg/authentication/request/websocket"
	"github.com/yubo/apiserver/pkg/authentication/token/bootstrap"
	tokencache "github.com/yubo/apiserver/pkg/authentication/token/cache"
	"github.com/yubo/apiserver/pkg/authentication/token/oidc"
	"github.com/yubo/apiserver/pkg/authentication/token/tokenfile"
	tokenunion "github.com/yubo/apiserver/pkg/authentication/token/union"
	"github.com/yubo/apiserver/pkg/listers"
	"github.com/yubo/apiserver/pkg/options"
	"github.com/yubo/apiserver/pkg/session"
	utilnet "github.com/yubo/golib/staging/util/net"
	"github.com/yubo/golib/staging/util/wait"
	"k8s.io/klog/v2"
)

// Config contains the data on how to authenticate a request to the Kube API Server
type Authentication struct {
	Anonymous                   bool
	BootstrapToken              bool
	TokenAuthFile               string
	OIDCIssuerURL               string
	OIDCClientID                string
	OIDCCAFile                  string
	OIDCUsernameClaim           string
	OIDCUsernamePrefix          string
	OIDCGroupsClaim             string
	OIDCGroupsPrefix            string
	OIDCSigningAlgs             []string
	OIDCRequiredClaims          map[string]string
	APIAudiences                authenticator.Audiences
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

func newAuthentication(ctx context.Context, c *config) (authn *Authentication, err error) {
	authn = &Authentication{
		TokenSuccessCacheTTL: c.TokenSuccessCacheTTL,
		TokenFailureCacheTTL: c.TokenFailureCacheTTL,
	}

	if c.Anonymous != nil {
		authn.Anonymous = c.Anonymous.Allow
	}

	if c.BootstrapToken != nil {
		authn.BootstrapToken = c.BootstrapToken.Enable
	}

	if c.OIDC != nil {
		authn.OIDCCAFile = c.OIDC.CAFile
		authn.OIDCClientID = c.OIDC.ClientID
		authn.OIDCGroupsClaim = c.OIDC.GroupsClaim
		authn.OIDCGroupsPrefix = c.OIDC.GroupsPrefix
		authn.OIDCIssuerURL = c.OIDC.IssuerURL
		authn.OIDCUsernameClaim = c.OIDC.UsernameClaim
		authn.OIDCUsernamePrefix = c.OIDC.UsernamePrefix
		authn.OIDCSigningAlgs = c.OIDC.SigningAlgs
		authn.OIDCRequiredClaims = c.OIDC.RequiredClaims
	}

	authn.APIAudiences = c.APIAudiences

	if c.TokenFile != nil {
		authn.TokenAuthFile = c.TokenFile.TokenFile
	}

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

	authn.Authenticator, err = authn.newAuthenticator(ctx)
	if err != nil {
		return nil, err
	}

	return authn, nil
}

// newAuthenticator returns an authenticator.Request or an error that supports the standard
// authentication mechanisms.
func (authn Authentication) newAuthenticator(ctx context.Context) (authenticator.Request, error) {
	var authenticators []authenticator.Request
	var tokenAuthenticators []authenticator.Token

	// TODO: deprecated, remove me
	if _, ok := options.SessionManagerFrom(ctx); ok {
		authenticators = append(authenticators, session.NewAuthenticator())
	}

	// Bearer token methods, local first, then remote
	if len(authn.TokenAuthFile) > 0 {
		tokenAuth, err := newAuthenticatorFromTokenFile(authn.TokenAuthFile)
		if err != nil {
			return nil, err
		}
		tokenAuthenticators = append(tokenAuthenticators,
			authenticator.WrapAudienceAgnosticToken(
				authn.APIAudiences,
				tokenAuth,
			))
	}
	if authn.BootstrapToken {
		authn.BootstrapTokenAuthenticator = bootstrap.NewTokenAuthenticator(
			listers.NewSecretLister(options.DBMustFrom(ctx)),
			//versionedInformer.Core().V1().Secrets().Lister().Secrets(metav1.NamespaceSystem),
		)

		tokenAuthenticators = append(tokenAuthenticators,
			authenticator.WrapAudienceAgnosticToken(
				authn.APIAudiences,
				authn.BootstrapTokenAuthenticator,
			))
	}
	// NOTE(ericchiang): Keep the OpenID Connect after Service Accounts.
	//
	// Because both plugins verify JWTs whichever comes first in the union experiences
	// cache misses for all requests using the other. While the service account plugin
	// simply returns an error, the OpenID Connect plugin may query the provider to
	// update the keys, causing performance hits.
	if len(authn.OIDCIssuerURL) > 0 && len(authn.OIDCClientID) > 0 {
		oidcAuth, err := newAuthenticatorFromOIDCIssuerURL(oidc.Options{
			IssuerURL:            authn.OIDCIssuerURL,
			ClientID:             authn.OIDCClientID,
			CAFile:               authn.OIDCCAFile,
			UsernameClaim:        authn.OIDCUsernameClaim,
			UsernamePrefix:       authn.OIDCUsernamePrefix,
			GroupsClaim:          authn.OIDCGroupsClaim,
			GroupsPrefix:         authn.OIDCGroupsPrefix,
			SupportedSigningAlgs: authn.OIDCSigningAlgs,
			RequiredClaims:       authn.OIDCRequiredClaims,
		})
		if err != nil {
			return nil, err
		}
		tokenAuthenticators = append(tokenAuthenticators,
			authenticator.WrapAudienceAgnosticToken(
				authn.APIAudiences,
				oidcAuth,
			))
	}

	if len(tokenAuthenticators) > 0 {
		// Union the token authenticators
		tokenAuth := tokenunion.New(tokenAuthenticators...)
		// Optionally cache authentication results
		if authn.TokenSuccessCacheTTL > 0 || authn.TokenFailureCacheTTL > 0 {
			tokenAuth = tokencache.New(tokenAuth, true,
				authn.TokenSuccessCacheTTL, authn.TokenFailureCacheTTL)
		}
		authenticators = append(authenticators,
			bearertoken.New(tokenAuth),
			websocket.NewProtocolAuthenticator(tokenAuth),
		)
	}

	if len(authenticators) == 0 {
		if authn.Anonymous {
			return anonymous.NewAuthenticator(), nil
		}
		return nil, nil
	}

	authenticator := union.New(authenticators...)

	authenticator = group.NewAuthenticatedGroupAdder(authenticator)

	if authn.Anonymous {
		// If the authenticator chain returns an error, return an error (don't consider a bad bearer token
		// or invalid username/password combination anonymous).
		authenticator = union.NewFailOnError(authenticator, anonymous.NewAuthenticator())
		klog.Infof("add anonymous authenticator")
	}

	return authenticator, nil
}

// newAuthenticatorFromTokenFile returns an authenticator.Token or an error
func newAuthenticatorFromTokenFile(tokenAuthFile string) (authenticator.Token, error) {
	tokenAuthenticator, err := tokenfile.NewCSV(tokenAuthFile)
	if err != nil {
		return nil, err
	}

	return tokenAuthenticator, nil
}

// newAuthenticatorFromOIDCIssuerURL returns an authenticator.Token or an error.
func newAuthenticatorFromOIDCIssuerURL(opts oidc.Options) (authenticator.Token, error) {
	const noUsernamePrefix = "-"

	if opts.UsernamePrefix == "" && opts.UsernameClaim != "email" {
		// Old behavior. If a usernamePrefix isn't provided, prefix all claims other than "email"
		// with the issuerURL.
		//
		// See https://github.com/kubernetes/kubernetes/issues/31380
		opts.UsernamePrefix = opts.IssuerURL + "#"
	}

	if opts.UsernamePrefix == noUsernamePrefix {
		// Special value indicating usernames shouldn't be prefixed.
		opts.UsernamePrefix = ""
	}

	tokenAuthenticator, err := oidc.New(opts)
	if err != nil {
		return nil, err
	}

	return tokenAuthenticator, nil
}
