/*
Copyright 2016 The Kubernetes Authors.

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

// from k8s.io/apiserver/pkg/server/options/authentication.go
package module

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/spf13/pflag"
	cliflag "github.com/yubo/golib/staging/cli/flag"
	utilerrors "github.com/yubo/golib/staging/util/errors"
	"github.com/yubo/golib/staging/util/wait"
	"github.com/yubo/golib/util"
)

// config contains all authentication options for API Server
type config struct {
	APIAudiences    []string                             `yaml:"apiAudiences"`
	Anonymous       *AnonymousAuthenticationOptions      `yaml:"anonymous"`
	BootstrapToken  *BootstrapTokenAuthenticationOptions `yaml:"bootstrapToken"`
	OIDC            *OIDCAuthenticationOptions           `yaml:"OIDC"`
	ServiceAccounts *ServiceAccountAuthenticationOptions
	TokenFile       *TokenFileAuthenticationOptions
	WebHook         *WebHookAuthenticationOptions

	TokenSuccessCacheTTL time.Duration
	TokenFailureCacheTTL time.Duration
}

// AnonymousAuthenticationOptions contains anonymous authentication options for API Server
type AnonymousAuthenticationOptions struct {
	Allow bool
}

// BootstrapTokenAuthenticationOptions contains bootstrap token authentication options for API Server
type BootstrapTokenAuthenticationOptions struct {
	Enable bool
}

// OIDCAuthenticationOptions contains OIDC authentication options for API Server
type OIDCAuthenticationOptions struct {
	CAFile         string
	ClientID       string
	IssuerURL      string
	UsernameClaim  string
	UsernamePrefix string
	GroupsClaim    string
	GroupsPrefix   string
	SigningAlgs    []string
	RequiredClaims map[string]string
}

// ServiceAccountAuthenticationOptions contains service account authentication options for API Server
type ServiceAccountAuthenticationOptions struct {
	KeyFiles         []string
	Lookup           bool
	Issuer           string
	JWKSURI          string
	MaxExpiration    time.Duration
	ExtendExpiration bool
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
	return &config{
		TokenSuccessCacheTTL: 10 * time.Second,
		TokenFailureCacheTTL: 0 * time.Second,
	}
}

func defaultConfig() *config {
	return newConfig().WithAll()
}

func (o *config) changed() interface{} {
	if o == nil {
		return nil
	}
	return util.Diff2Map(defaultConfig(), o)
}

// WithAll set default value for every build-in authentication option
func (o *config) WithAll() *config {
	return o.
		WithAnonymous().
		WithBootstrapToken().
		//WithClientCert().
		WithOIDC().
		//WithServiceAccounts().
		WithTokenFile().
		WithWebHook()
}

// WithAnonymous set default value for anonymous authentication
func (o *config) WithAnonymous() *config {
	o.Anonymous = &AnonymousAuthenticationOptions{Allow: true}
	return o
}

// WithBootstrapToken set default value for bootstrap token authentication
func (o *config) WithBootstrapToken() *config {
	o.BootstrapToken = &BootstrapTokenAuthenticationOptions{}
	return o
}

// WithClientCert set default value for client cert
//func (o *BuiltInAuthenticationOptions) WithClientCert() *BuiltInAuthenticationOptions {
//	o.ClientCert = &ClientCertAuthenticationOptions{}
//	return o
//}

// WithOIDC set default value for OIDC authentication
func (o *config) WithOIDC() *config {
	o.OIDC = &OIDCAuthenticationOptions{}
	return o
}

// WithServiceAccounts set default value for service account authentication
func (o *config) WithServiceAccounts() *config {
	o.ServiceAccounts = &ServiceAccountAuthenticationOptions{Lookup: true, ExtendExpiration: true}
	return o
}

// WithTokenFile set default value for token file authentication
func (o *config) WithTokenFile() *config {
	o.TokenFile = &TokenFileAuthenticationOptions{}
	return o
}

// WithWebHook set default value for web hook authentication
func (o *config) WithWebHook() *config {
	o.WebHook = &WebHookAuthenticationOptions{
		Version:      "v1beta1",
		CacheTTL:     2 * time.Minute,
		RetryBackoff: DefaultAuthWebhookRetryBackoff(),
	}
	return o
}

// Validate checks invalid config combination
func (o *config) Validate() error {
	allErrors := []error{}

	if o.OIDC != nil && (len(o.OIDC.IssuerURL) > 0) != (len(o.OIDC.ClientID) > 0) {
		allErrors = append(allErrors, fmt.Errorf("oidc-issuer-url and oidc-client-id should be specified together"))
	}

	if o.ServiceAccounts != nil && len(o.ServiceAccounts.Issuer) > 0 && strings.Contains(o.ServiceAccounts.Issuer, ":") {
		if _, err := url.Parse(o.ServiceAccounts.Issuer); err != nil {
			allErrors = append(allErrors, fmt.Errorf("service-account-issuer contained a ':' but was not a valid URL: %v", err))
		}
	}

	if o.ServiceAccounts != nil {
		if len(o.ServiceAccounts.Issuer) == 0 {
			allErrors = append(allErrors, errors.New("service-account-issuer is a required flag"))
		}
		if len(o.ServiceAccounts.KeyFiles) == 0 {
			allErrors = append(allErrors, errors.New("service-account-key-file is a required flag"))
		}

		// Validate the JWKS URI when it is explicitly set.
		// When unset, it is later derived from ExternalHost.
		if o.ServiceAccounts.JWKSURI != "" {
			if u, err := url.Parse(o.ServiceAccounts.JWKSURI); err != nil {
				allErrors = append(allErrors, fmt.Errorf("service-account-jwks-uri must be a valid URL: %v", err))
			} else if u.Scheme != "https" {
				allErrors = append(allErrors, fmt.Errorf("service-account-jwks-uri requires https scheme, parsed as: %v", u.String()))
			}
		}
	}

	if o.WebHook != nil {
		retryBackoff := o.WebHook.RetryBackoff
		if retryBackoff != nil && retryBackoff.Steps <= 0 {
			allErrors = append(allErrors, fmt.Errorf("number of webhook retry attempts must be greater than 1, but is: %d", retryBackoff.Steps))
		}
	}

	return utilerrors.NewAggregate(allErrors)
}

// addFlags returns flags of authentication for a API Server
func (o *config) addFlags(fs *pflag.FlagSet) {
	fs.StringSliceVar(&o.APIAudiences, "api-audiences", o.APIAudiences, ""+
		"Identifiers of the API. The service account token authenticator will validate that "+
		"tokens used against the API are bound to at least one of these audiences. If the "+
		"--service-account-issuer flag is configured and this flag is not, this field "+
		"defaults to a single element list containing the issuer URL.")

	if o.Anonymous != nil {
		fs.BoolVar(&o.Anonymous.Allow, "anonymous-auth", o.Anonymous.Allow, ""+
			"Enables anonymous requests to the secure port of the API server. "+
			"Requests that are not rejected by another authentication method are treated as anonymous requests. "+
			"Anonymous requests have a username of system:anonymous, and a group name of system:unauthenticated.")
	}

	if o.BootstrapToken != nil {
		fs.BoolVar(&o.BootstrapToken.Enable, "enable-bootstrap-token-auth", o.BootstrapToken.Enable, ""+
			"Enable to allow secrets of type 'bootstrap.kubernetes.io/token' in the 'kube-system' "+
			"namespace to be used for TLS bootstrapping authentication.")
	}

	//if o.ClientCert != nil {
	//	o.ClientCert.AddFlags(fs)
	//}

	if o.OIDC != nil {
		fs.StringVar(&o.OIDC.IssuerURL, "oidc-issuer-url", o.OIDC.IssuerURL, ""+
			"The URL of the OpenID issuer, only HTTPS scheme will be accepted. "+
			"If set, it will be used to verify the OIDC JSON Web Token (JWT).")

		fs.StringVar(&o.OIDC.ClientID, "oidc-client-id", o.OIDC.ClientID,
			"The client ID for the OpenID Connect client, must be set if oidc-issuer-url is set.")

		fs.StringVar(&o.OIDC.CAFile, "oidc-ca-file", o.OIDC.CAFile, ""+
			"If set, the OpenID server's certificate will be verified by one of the authorities "+
			"in the oidc-ca-file, otherwise the host's root CA set will be used.")

		fs.StringVar(&o.OIDC.UsernameClaim, "oidc-username-claim", "sub", ""+
			"The OpenID claim to use as the user name. Note that claims other than the default ('sub') "+
			"is not guaranteed to be unique and immutable. This flag is experimental, please see "+
			"the authentication documentation for further details.")

		fs.StringVar(&o.OIDC.UsernamePrefix, "oidc-username-prefix", "", ""+
			"If provided, all usernames will be prefixed with this value. If not provided, "+
			"username claims other than 'email' are prefixed by the issuer URL to avoid "+
			"clashes. To skip any prefixing, provide the value '-'.")

		fs.StringVar(&o.OIDC.GroupsClaim, "oidc-groups-claim", "", ""+
			"If provided, the name of a custom OpenID Connect claim for specifying user groups. "+
			"The claim value is expected to be a string or array of strings. This flag is experimental, "+
			"please see the authentication documentation for further details.")

		fs.StringVar(&o.OIDC.GroupsPrefix, "oidc-groups-prefix", "", ""+
			"If provided, all groups will be prefixed with this value to prevent conflicts with "+
			"other authentication strategies.")

		fs.StringSliceVar(&o.OIDC.SigningAlgs, "oidc-signing-algs", []string{"RS256"}, ""+
			"Comma-separated list of allowed JOSE asymmetric signing algorithms. JWTs with a "+
			"'alg' header value not in this list will be rejected. "+
			"Values are defined by RFC 7518 https://tools.ietf.org/html/rfc7518#section-3.1.")

		fs.Var(cliflag.NewMapStringStringNoSplit(&o.OIDC.RequiredClaims), "oidc-required-claim", ""+
			"A key=value pair that describes a required claim in the ID Token. "+
			"If set, the claim is verified to be present in the ID Token with a matching value. "+
			"Repeat this flag to specify multiple claims.")
	}

	/*
		if o.RequestHeader != nil {
			o.RequestHeader.AddFlags(fs)
		}
	*/

	if o.ServiceAccounts != nil {
		fs.StringArrayVar(&o.ServiceAccounts.KeyFiles, "service-account-key-file", o.ServiceAccounts.KeyFiles, ""+
			"File containing PEM-encoded x509 RSA or ECDSA private or public keys, used to verify "+
			"ServiceAccount tokens. The specified file can contain multiple keys, and the flag can "+
			"be specified multiple times with different files. If unspecified, "+
			"--tls-private-key-file is used. Must be specified when "+
			"--service-account-signing-key is provided")

		fs.BoolVar(&o.ServiceAccounts.Lookup, "service-account-lookup", o.ServiceAccounts.Lookup,
			"If true, validate ServiceAccount tokens exist in etcd as part of authentication.")

		fs.StringVar(&o.ServiceAccounts.Issuer, "service-account-issuer", o.ServiceAccounts.Issuer, ""+
			"Identifier of the service account token issuer. The issuer will assert this identifier "+
			"in \"iss\" claim of issued tokens. This value is a string or URI. If this option is not "+
			"a valid URI per the OpenID Discovery 1.0 spec, the ServiceAccountIssuerDiscovery feature "+
			"will remain disabled, even if the feature gate is set to true. It is highly recommended "+
			"that this value comply with the OpenID spec: https://openid.net/specs/openid-connect-discovery-1_0.html. "+
			"In practice, this means that service-account-issuer must be an https URL. It is also highly "+
			"recommended that this URL be capable of serving OpenID discovery documents at "+
			"{service-account-issuer}/.well-known/openid-configuration.")

		fs.StringVar(&o.ServiceAccounts.JWKSURI, "service-account-jwks-uri", o.ServiceAccounts.JWKSURI, ""+
			"Overrides the URI for the JSON Web Key Set in the discovery doc served at "+
			"/.well-known/openid-configuration. This flag is useful if the discovery doc"+
			"and key set are served to relying parties from a URL other than the "+
			"API server's external (as auto-detected or overridden with external-hostname). "+
			"Only valid if the ServiceAccountIssuerDiscovery feature gate is enabled.")

		// Deprecated in 1.13
		fs.StringSliceVar(&o.APIAudiences, "service-account-api-audiences", o.APIAudiences, ""+
			"Identifiers of the API. The service account token authenticator will validate that "+
			"tokens used against the API are bound to at least one of these audiences.")
		fs.MarkDeprecated("service-account-api-audiences", "Use --api-audiences")

		fs.DurationVar(&o.ServiceAccounts.MaxExpiration, "service-account-max-token-expiration", o.ServiceAccounts.MaxExpiration, ""+
			"The maximum validity duration of a token created by the service account token issuer. If an otherwise valid "+
			"TokenRequest with a validity duration larger than this value is requested, a token will be issued with a validity duration of this value.")

		fs.BoolVar(&o.ServiceAccounts.ExtendExpiration, "service-account-extend-token-expiration", o.ServiceAccounts.ExtendExpiration, ""+
			"Turns on projected service account expiration extension during token generation, "+
			"which helps safe transition from legacy token to bound service account token feature. "+
			"If this flag is enabled, admission injected tokens would be extended up to 1 year to "+
			"prevent unexpected failure during transition, ignoring value of service-account-max-token-expiration.")
	}

	if o.TokenFile != nil {
		fs.StringVar(&o.TokenFile.TokenFile, "token-auth-file", o.TokenFile.TokenFile, ""+
			"If set, the file that will be used to secure the secure port of the API server "+
			"via token authentication.")
	}

	if o.WebHook != nil {
		fs.StringVar(&o.WebHook.ConfigFile, "authentication-token-webhook-config-file", o.WebHook.ConfigFile, ""+
			"File with webhook configuration for token authentication in kubeconfig format. "+
			"The API server will query the remote service to determine authentication for bearer tokens.")

		fs.StringVar(&o.WebHook.Version, "authentication-token-webhook-version", o.WebHook.Version, ""+
			"The API version of the authentication.k8s.io TokenReview to send to and expect from the webhook.")

		fs.DurationVar(&o.WebHook.CacheTTL, "authentication-token-webhook-cache-ttl", o.WebHook.CacheTTL,
			"The duration to cache responses from the webhook token authenticator.")
	}
}

// DefaultAuthWebhookRetryBackoff is the default backoff parameters for
// both authentication and authorization webhook used by the apiserver.
func DefaultAuthWebhookRetryBackoff() *wait.Backoff {
	return &wait.Backoff{
		Duration: 500 * time.Millisecond,
		Factor:   1.5,
		Jitter:   0.2,
		Steps:    5,
	}
}
