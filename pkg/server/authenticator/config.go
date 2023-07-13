/*
Copyright 2014 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package authenticator

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/yubo/apiserver/pkg/authentication"
	"github.com/yubo/apiserver/pkg/authentication/authenticator"
	"github.com/yubo/apiserver/pkg/authentication/authenticatorfactory"
	"github.com/yubo/apiserver/pkg/authentication/group"
	"github.com/yubo/apiserver/pkg/authentication/request/anonymous"
	"github.com/yubo/apiserver/pkg/authentication/request/bearertoken"
	"github.com/yubo/apiserver/pkg/authentication/request/headerrequest"
	"github.com/yubo/apiserver/pkg/authentication/request/union"
	"github.com/yubo/apiserver/pkg/authentication/request/websocket"
	"github.com/yubo/apiserver/pkg/authentication/request/x509"
	tokencache "github.com/yubo/apiserver/pkg/authentication/token/cache"
	"github.com/yubo/apiserver/pkg/authentication/token/tokenfile"
	tokenunion "github.com/yubo/apiserver/pkg/authentication/token/union"
	"github.com/yubo/apiserver/pkg/server/dynamiccertificates"
	webhookutil "github.com/yubo/apiserver/pkg/util/webhook"
	"github.com/yubo/apiserver/plugin/authenticator/token/oidc"
	"github.com/yubo/apiserver/plugin/authenticator/token/webhook"
	"github.com/yubo/golib/util"
	utilnet "github.com/yubo/golib/util/net"
	"github.com/yubo/golib/util/wait"

	// Initialize all known client auth plugins.
	_ "github.com/yubo/client-go/plugin/pkg/client/auth"
	"github.com/yubo/client-go/util/keyutil"
	//"k8s.io/kubernetes/pkg/serviceaccount"
)

// Config contains the data on how to authenticate a request to the Kube API Server
type Config struct {
	Anonymous      bool
	BootstrapToken bool

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
	ServiceAccountKeyFiles      []string
	ServiceAccountLookup        bool
	ServiceAccountIssuers       []string
	APIAudiences                authenticator.Audiences
	WebhookTokenAuthnConfigFile string
	WebhookTokenAuthnVersion    string
	WebhookTokenAuthnCacheTTL   time.Duration
	// WebhookRetryBackoff specifies the backoff parameters for the authentication webhook retry logic.
	// This allows us to configure the sleep time at each iteration and the maximum number of retries allowed
	// before we fail the webhook call in order to limit the fan out that ensues when the system is degraded.
	WebhookRetryBackoff *wait.Backoff

	TokenSuccessCacheTTL time.Duration
	TokenFailureCacheTTL time.Duration

	RequestHeaderConfig *authenticatorfactory.RequestHeaderConfig

	// TODO, this is the only non-serializable part of the entire config.  Factor it out into a clientconfig
	//ServiceAccountTokenGetter   serviceaccount.ServiceAccountTokenGetter
	//SecretsWriter               typedv1core.SecretsGetter
	// move to custom plugin
	//BootstrapTokenAuthenticator authenticator.Token

	// ClientCAContentProvider are the options for verifying incoming connections using mTLS and directly assigning to users.
	// Generally this is the CA bundle file used to authenticate client certificates
	// If this value is nil, then mutual TLS is disabled.
	ClientCAContentProvider dynamiccertificates.CAContentProvider

	// Optional field, custom dial function used to connect to webhook
	CustomDial utilnet.DialFunc
}

// New returns an authenticator.Request or an error that supports the standard
// Kubernetes authentication mechanisms.
func (config Config) New(ctx context.Context) (authenticator.Request /* *spec.SecurityDefinitions, */, error) {
	var authenticators []authenticator.Request
	var tokenAuthenticators []authenticator.Token
	//securityDefinitions := spec.SecurityDefinitions{}

	// front-proxy, BasicAuth methods, local first, then remote
	// Add the front proxy authenticator if requested
	if config.RequestHeaderConfig != nil {
		requestHeaderAuthenticator := headerrequest.NewDynamicVerifyOptionsSecure(
			config.RequestHeaderConfig.CAContentProvider.VerifyOptions,
			config.RequestHeaderConfig.AllowedClientNames,
			config.RequestHeaderConfig.UsernameHeaders,
			config.RequestHeaderConfig.GroupHeaders,
			config.RequestHeaderConfig.ExtraHeaderPrefixes,
		)
		authenticators = append(authenticators, authenticator.WrapAudienceAgnosticRequest(config.APIAudiences, requestHeaderAuthenticator))
	}

	// X509 methods
	if config.ClientCAContentProvider != nil {
		certAuth := x509.NewDynamic(config.ClientCAContentProvider.VerifyOptions, x509.CommonNameUserConversion)
		authenticators = append(authenticators, certAuth)
	}

	// Bearer token methods, local first, then remote
	if len(config.TokenAuthFile) > 0 {
		tokenAuth, err := newAuthenticatorFromTokenFile(config.TokenAuthFile)
		if err != nil {
			return nil, err
		}
		tokenAuthenticators = append(tokenAuthenticators, authenticator.WrapAudienceAgnosticToken(config.APIAudiences, tokenAuth))
	}
	//if len(config.ServiceAccountKeyFiles) > 0 {
	//	serviceAccountAuth, err := newLegacyServiceAccountAuthenticator(config.ServiceAccountKeyFiles, config.ServiceAccountLookup, config.APIAudiences, config.ServiceAccountTokenGetter, config.SecretsWriter)
	//	if err != nil {
	//		return nil, err
	//	}
	//	tokenAuthenticators = append(tokenAuthenticators, serviceAccountAuth)
	//}
	//if len(config.ServiceAccountIssuers) > 0 {
	//	serviceAccountAuth, err := newServiceAccountAuthenticator(config.ServiceAccountIssuers, config.ServiceAccountKeyFiles, config.APIAudiences, config.ServiceAccountTokenGetter)
	//	if err != nil {
	//		return nil, err
	//	}
	//	tokenAuthenticators = append(tokenAuthenticators, serviceAccountAuth)
	//}

	//if config.BootstrapToken && config.BootstrapTokenAuthenticator != nil {
	//	tokenAuthenticators = append(tokenAuthenticators, authenticator.WrapAudienceAgnosticToken(config.APIAudiences, config.BootstrapTokenAuthenticator))
	//}

	// NOTE(ericchiang): Keep the OpenID Connect after Service Accounts.
	//
	// Because both plugins verify JWTs whichever comes first in the union experiences
	// cache misses for all requests using the other. While the service account plugin
	// simply returns an error, the OpenID Connect plugin may query the provider to
	// update the keys, causing performance hits.
	if len(config.OIDCIssuerURL) > 0 && len(config.OIDCClientID) > 0 {
		// TODO(enj): wire up the Notifier and ControllerRunner bits when OIDC supports CA reload
		var oidcCAContent oidc.CAContentProvider
		if len(config.OIDCCAFile) != 0 {
			var oidcCAErr error
			oidcCAContent, oidcCAErr = staticCAContentProviderFromFile("oidc-authenticator", config.OIDCCAFile)
			if oidcCAErr != nil {
				return nil, oidcCAErr
			}
		}

		oidcAuth, err := newAuthenticatorFromOIDCIssuerURL(oidc.Options{
			IssuerURL:            config.OIDCIssuerURL,
			ClientID:             config.OIDCClientID,
			CAContentProvider:    oidcCAContent,
			UsernameClaim:        config.OIDCUsernameClaim,
			UsernamePrefix:       config.OIDCUsernamePrefix,
			GroupsClaim:          config.OIDCGroupsClaim,
			GroupsPrefix:         config.OIDCGroupsPrefix,
			SupportedSigningAlgs: config.OIDCSigningAlgs,
			RequiredClaims:       config.OIDCRequiredClaims,
		})
		if err != nil {
			return nil, err
		}
		tokenAuthenticators = append(tokenAuthenticators, authenticator.WrapAudienceAgnosticToken(config.APIAudiences, oidcAuth))
	}
	if len(config.WebhookTokenAuthnConfigFile) > 0 {
		webhookTokenAuth, err := newWebhookTokenAuthenticator(config)
		if err != nil {
			return nil, err
		}

		tokenAuthenticators = append(tokenAuthenticators, webhookTokenAuth)
	}

	// custom authn
	for _, factory := range authentication.AuthenticatorFactories() {
		auth, err := factory(ctx)
		if err != nil {
			return nil, err
		}
		if !util.IsNil(auth) {
			authenticators = append(authenticators, auth)
		}
	}

	for _, factory := range authentication.TokenAuthenticatorFactories() {
		token, err := factory(ctx)
		if err != nil {
			return nil, err
		}
		if !util.IsNil(token) {
			tokenAuthenticators = append(tokenAuthenticators, token)
		}
	}

	if len(tokenAuthenticators) > 0 {
		// Union the token authenticators
		tokenAuth := tokenunion.New(tokenAuthenticators...)
		// Optionally cache authentication results
		if config.TokenSuccessCacheTTL > 0 || config.TokenFailureCacheTTL > 0 {
			tokenAuth = tokencache.New(tokenAuth, true, config.TokenSuccessCacheTTL, config.TokenFailureCacheTTL)
		}
		authenticators = append(authenticators, bearertoken.New(tokenAuth), websocket.NewProtocolAuthenticator(tokenAuth))
		//securityDefinitions["BearerToken"] = &spec.SecurityScheme{
		//	SecuritySchemeProps: spec.SecuritySchemeProps{
		//		Type:        "apiKey",
		//		Name:        "authorization",
		//		In:          "header",
		//		Description: "Bearer Token authentication",
		//	},
		//}
	}

	if len(authenticators) == 0 {
		if config.Anonymous {
			return anonymous.NewAuthenticator(), nil
		}
		return nil, nil
	}

	authenticator := union.New(authenticators...)

	authenticator = group.NewAuthenticatedGroupAdder(authenticator)

	if config.Anonymous {
		// If the authenticator chain returns an error, return an error (don't consider a bad bearer token
		// or invalid username/password combination anonymous).
		authenticator = union.NewFailOnError(authenticator, anonymous.NewAuthenticator())
	}

	return authenticator, nil
}

// IsValidServiceAccountKeyFile returns true if a valid public RSA key can be read from the given file
func IsValidServiceAccountKeyFile(file string) bool {
	_, err := keyutil.PublicKeysFromFile(file)
	return err == nil
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

// newLegacyServiceAccountAuthenticator returns an authenticator.Token or an error
//func newLegacyServiceAccountAuthenticator(keyfiles []string, lookup bool, apiAudiences authenticator.Audiences, serviceAccountGetter serviceaccount.ServiceAccountTokenGetter, secretsWriter typedv1core.SecretsGetter) (authenticator.Token, error) {
//	allPublicKeys := []interface{}{}
//	for _, keyfile := range keyfiles {
//		publicKeys, err := keyutil.PublicKeysFromFile(keyfile)
//		if err != nil {
//			return nil, err
//		}
//		allPublicKeys = append(allPublicKeys, publicKeys...)
//	}
//	validator, err := serviceaccount.NewLegacyValidator(lookup, serviceAccountGetter, secretsWriter)
//	if err != nil {
//		return nil, fmt.Errorf("while creating legacy validator, err: %w", err)
//	}
//
//	tokenAuthenticator := serviceaccount.JWTTokenAuthenticator([]string{serviceaccount.LegacyIssuer}, allPublicKeys, apiAudiences, validator)
//	return tokenAuthenticator, nil
//}

// newServiceAccountAuthenticator returns an authenticator.Token or an error
//func newServiceAccountAuthenticator(issuers []string, keyfiles []string, apiAudiences authenticator.Audiences, serviceAccountGetter serviceaccount.ServiceAccountTokenGetter) (authenticator.Token, error) {
//	allPublicKeys := []interface{}{}
//	for _, keyfile := range keyfiles {
//		publicKeys, err := keyutil.PublicKeysFromFile(keyfile)
//		if err != nil {
//			return nil, err
//		}
//		allPublicKeys = append(allPublicKeys, publicKeys...)
//	}
//
//	tokenAuthenticator := serviceaccount.JWTTokenAuthenticator(issuers, allPublicKeys, apiAudiences, serviceaccount.NewValidator(serviceAccountGetter))
//	return tokenAuthenticator, nil
//}

func newWebhookTokenAuthenticator(config Config) (authenticator.Token, error) {
	if config.WebhookRetryBackoff == nil {
		return nil, errors.New("retry backoff parameters for authentication webhook has not been specified")
	}

	clientConfig, err := webhookutil.LoadKubeconfig(config.WebhookTokenAuthnConfigFile, config.CustomDial)
	if err != nil {
		return nil, err
	}
	webhookTokenAuthenticator, err := webhook.New(clientConfig /*config.WebhookTokenAuthnVersion,*/, config.APIAudiences, *config.WebhookRetryBackoff)
	if err != nil {
		return nil, err
	}

	return tokencache.New(webhookTokenAuthenticator, false, config.WebhookTokenAuthnCacheTTL, config.WebhookTokenAuthnCacheTTL), nil
}

func staticCAContentProviderFromFile(purpose, filename string) (dynamiccertificates.CAContentProvider, error) {
	fileBytes, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	return dynamiccertificates.NewStaticCAContent(purpose, fileBytes)
}