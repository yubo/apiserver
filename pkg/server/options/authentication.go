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

package options

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/yubo/golib/api"

	"github.com/yubo/apiserver/pkg/authentication/authenticatorfactory"
	"github.com/yubo/apiserver/pkg/authentication/request/headerrequest"
	authzmodes "github.com/yubo/apiserver/pkg/authorization/authorizer/modes"
	"github.com/yubo/apiserver/pkg/models"
	genericapiserver "github.com/yubo/apiserver/pkg/server"
	serverauthenticator "github.com/yubo/apiserver/pkg/server/authenticator"
	"github.com/yubo/apiserver/pkg/server/dynamiccertificates"
	"github.com/yubo/apiserver/plugin/authenticator/token/bootstrap"
	"github.com/yubo/golib/util/sets"
	"github.com/yubo/golib/util/wait"
	"k8s.io/klog/v2"
)

// #################### kubeapiserver/options/authentication.go

// BuiltInAuthenticationOptions contains all build-in authentication options for API Server
type BuiltInAuthenticationOptions struct {
	APIAudiences   []string                        `json:"apiAudiences" flag:"api-audiences" description:"Identifiers of the API. The service account token authenticator will validate that tokens used against the API are bound to at least one of these audiences. If the --service-account-issuer flag is configured and this flag is not, this field defaults to a single element list containing the issuer URL."`
	Anonymous      *AnonymousAuthenticationOptions `json:"inline"`
	BootstrapToken *BootstrapTokenAuthenticationOptions
	ClientCert     *ClientCertAuthenticationOptions
	OIDC           *OIDCAuthenticationOptions
	RequestHeader  *RequestHeaderAuthenticationOptions
	//ServiceAccounts *ServiceAccountAuthenticationOptions
	TokenFile *TokenFileAuthenticationOptions
	WebHook   *WebHookAuthenticationOptions

	TokenSuccessCacheTTL api.Duration `json:"tokenSuccessCacheTTL" flag:"token-success-cache-ttl" default:"10s" description:"The duration to cache success token."`
	TokenFailureCacheTTL api.Duration `json:"tokenFailureCacheTTL" flag:"token-failure-cache-ttl" description:"The duration to cache failure token."`
}

// AnonymousAuthenticationOptions contains anonymous authentication options for API Server
type AnonymousAuthenticationOptions struct {
	Allow bool `json:"anonymous" flag:"anonymous-auth" default:"false" description:"Enables anonymous requests to the secure port of the API server. Requests that are not rejected by another authentication method are treated as anonymous requests. Anonymous requests have a username of system:anonymous, and a group name of system:unauthenticated."`
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
	Issuers          []string
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

// NewBuiltInAuthenticationOptions create a new BuiltInAuthenticationOptions, just set default token cache TTL
func NewBuiltInAuthenticationOptions() *BuiltInAuthenticationOptions {
	return &BuiltInAuthenticationOptions{
		TokenSuccessCacheTTL: api.NewDuration("10s"),
		TokenFailureCacheTTL: api.NewDuration("0s"),
	}
}

// WithAll set default value for every build-in authentication option
func (o *BuiltInAuthenticationOptions) WithAll() *BuiltInAuthenticationOptions {
	return o.
		WithAnonymous().
		WithBootstrapToken().
		WithClientCert().
		WithOIDC().
		WithRequestHeader().
		//WithServiceAccounts().
		WithTokenFile().
		WithWebHook()
}

// WithAnonymous set default value for anonymous authentication
func (o *BuiltInAuthenticationOptions) WithAnonymous() *BuiltInAuthenticationOptions {
	o.Anonymous = &AnonymousAuthenticationOptions{Allow: true}
	return o
}

// WithBootstrapToken set default value for bootstrap token authentication
func (o *BuiltInAuthenticationOptions) WithBootstrapToken() *BuiltInAuthenticationOptions {
	o.BootstrapToken = &BootstrapTokenAuthenticationOptions{}
	return o
}

// WithClientCert set default value for client cert
func (o *BuiltInAuthenticationOptions) WithClientCert() *BuiltInAuthenticationOptions {
	o.ClientCert = &ClientCertAuthenticationOptions{}
	return o
}

// WithOIDC set default value for OIDC authentication
func (o *BuiltInAuthenticationOptions) WithOIDC() *BuiltInAuthenticationOptions {
	o.OIDC = &OIDCAuthenticationOptions{}
	return o
}

// WithRequestHeader set default value for request header authentication
func (o *BuiltInAuthenticationOptions) WithRequestHeader() *BuiltInAuthenticationOptions {
	o.RequestHeader = &RequestHeaderAuthenticationOptions{}
	return o
}

// WithServiceAccounts set default value for service account authentication
//func (o *BuiltInAuthenticationOptions) WithServiceAccounts() *BuiltInAuthenticationOptions {
//	o.ServiceAccounts = &ServiceAccountAuthenticationOptions{Lookup: true, ExtendExpiration: true}
//	return o
//}

// WithTokenFile set default value for token file authentication
func (o *BuiltInAuthenticationOptions) WithTokenFile() *BuiltInAuthenticationOptions {
	o.TokenFile = &TokenFileAuthenticationOptions{}
	return o
}

// WithWebHook set default value for web hook authentication
func (o *BuiltInAuthenticationOptions) WithWebHook() *BuiltInAuthenticationOptions {
	o.WebHook = &WebHookAuthenticationOptions{
		Version:      "v1beta1",
		CacheTTL:     2 * time.Minute,
		RetryBackoff: DefaultAuthWebhookRetryBackoff(),
	}
	return o
}

// Validate checks invalid config combination
func (o *BuiltInAuthenticationOptions) Validate() []error {
	var allErrors []error

	if o.OIDC != nil && (len(o.OIDC.IssuerURL) > 0) != (len(o.OIDC.ClientID) > 0) {
		allErrors = append(allErrors, fmt.Errorf("oidc-issuer-url and oidc-client-id should be specified together"))
	}

	//if o.ServiceAccounts != nil && len(o.ServiceAccounts.Issuers) > 0 {
	//	seen := make(map[string]bool)
	//	for _, issuer := range o.ServiceAccounts.Issuers {
	//		if strings.Contains(issuer, ":") {
	//			if _, err := url.Parse(issuer); err != nil {
	//				allErrors = append(allErrors, fmt.Errorf("service-account-issuer %q contained a ':' but was not a valid URL: %v", issuer, err))
	//				continue
	//			}
	//		}
	//		if issuer == "" {
	//			allErrors = append(allErrors, fmt.Errorf("service-account-issuer should not be an empty string"))
	//			continue
	//		}
	//		if seen[issuer] {
	//			allErrors = append(allErrors, fmt.Errorf("service-account-issuer %q is already specified", issuer))
	//			continue
	//		}
	//		seen[issuer] = true
	//	}
	//}

	//if o.ServiceAccounts != nil {
	//	if len(o.ServiceAccounts.Issuers) == 0 {
	//		allErrors = append(allErrors, errors.New("service-account-issuer is a required flag"))
	//	}
	//	if len(o.ServiceAccounts.KeyFiles) == 0 {
	//		allErrors = append(allErrors, errors.New("service-account-key-file is a required flag"))
	//	}

	//	// Validate the JWKS URI when it is explicitly set.
	//	// When unset, it is later derived from ExternalHost.
	//	if o.ServiceAccounts.JWKSURI != "" {
	//		if u, err := url.Parse(o.ServiceAccounts.JWKSURI); err != nil {
	//			allErrors = append(allErrors, fmt.Errorf("service-account-jwks-uri must be a valid URL: %v", err))
	//		} else if u.Scheme != "https" {
	//			allErrors = append(allErrors, fmt.Errorf("service-account-jwks-uri requires https scheme, parsed as: %v", u.String()))
	//		}
	//	}
	//}

	if o.WebHook != nil {
		retryBackoff := o.WebHook.RetryBackoff
		if retryBackoff != nil && retryBackoff.Steps <= 0 {
			allErrors = append(allErrors, fmt.Errorf("number of webhook retry attempts must be greater than 0, but is: %d", retryBackoff.Steps))
		}
	}

	if o.RequestHeader != nil {
		allErrors = append(allErrors, o.RequestHeader.Validate()...)
	}

	return allErrors
}

// ToAuthenticationConfig convert BuiltInAuthenticationOptions to serverauthenticator.Config
func (o *BuiltInAuthenticationOptions) ToAuthenticationConfig() (serverauthenticator.Config, error) {
	ret := serverauthenticator.Config{
		TokenSuccessCacheTTL: o.TokenSuccessCacheTTL.Duration,
		TokenFailureCacheTTL: o.TokenFailureCacheTTL.Duration,
	}

	if o.Anonymous != nil {
		ret.Anonymous = o.Anonymous.Allow
	}

	if o.BootstrapToken != nil {
		ret.BootstrapToken = o.BootstrapToken.Enable
	}

	if o.ClientCert != nil {
		var err error
		ret.ClientCAContentProvider, err = o.ClientCert.GetClientCAContentProvider()
		if err != nil {
			return serverauthenticator.Config{}, err
		}
	}

	if o.OIDC != nil {
		ret.OIDCCAFile = o.OIDC.CAFile
		ret.OIDCClientID = o.OIDC.ClientID
		ret.OIDCGroupsClaim = o.OIDC.GroupsClaim
		ret.OIDCGroupsPrefix = o.OIDC.GroupsPrefix
		ret.OIDCIssuerURL = o.OIDC.IssuerURL
		ret.OIDCUsernameClaim = o.OIDC.UsernameClaim
		ret.OIDCUsernamePrefix = o.OIDC.UsernamePrefix
		ret.OIDCSigningAlgs = o.OIDC.SigningAlgs
		ret.OIDCRequiredClaims = o.OIDC.RequiredClaims
	}

	if o.RequestHeader != nil {
		var err error
		ret.RequestHeaderConfig, err = o.RequestHeader.ToAuthenticationRequestHeaderConfig()
		if err != nil {
			return serverauthenticator.Config{}, err
		}
	}

	ret.APIAudiences = o.APIAudiences
	//if o.ServiceAccounts != nil {
	//	if len(o.ServiceAccounts.Issuers) != 0 && len(o.APIAudiences) == 0 {
	//		ret.APIAudiences = authenticator.Audiences(o.ServiceAccounts.Issuers)
	//	}
	//	ret.ServiceAccountKeyFiles = o.ServiceAccounts.KeyFiles
	//	ret.ServiceAccountIssuers = o.ServiceAccounts.Issuers
	//	ret.ServiceAccountLookup = o.ServiceAccounts.Lookup
	//}

	if o.TokenFile != nil {
		ret.TokenAuthFile = o.TokenFile.TokenFile
	}

	if o.WebHook != nil {
		ret.WebhookTokenAuthnConfigFile = o.WebHook.ConfigFile
		ret.WebhookTokenAuthnVersion = o.WebHook.Version
		ret.WebhookTokenAuthnCacheTTL = o.WebHook.CacheTTL
		ret.WebhookRetryBackoff = o.WebHook.RetryBackoff

		if len(o.WebHook.ConfigFile) > 0 && o.WebHook.CacheTTL > 0 {
			if o.TokenSuccessCacheTTL.Duration > 0 && o.WebHook.CacheTTL < o.TokenSuccessCacheTTL.Duration {
				klog.Warningf("the webhook cache ttl of %s is shorter than the overall cache ttl of %s for successful token authentication attempts.", o.WebHook.CacheTTL, o.TokenSuccessCacheTTL)
			}
			if o.TokenFailureCacheTTL.Duration > 0 && o.WebHook.CacheTTL < o.TokenFailureCacheTTL.Duration {
				klog.Warningf("the webhook cache ttl of %s is shorter than the overall cache ttl of %s for failed token authentication attempts.", o.WebHook.CacheTTL, o.TokenFailureCacheTTL)
			}
		}
	}

	return ret, nil
}

// ApplyTo requires already applied OpenAPIConfig and EgressSelector if present.
func (o *BuiltInAuthenticationOptions) ApplyTo(ctx context.Context, authInfo *genericapiserver.AuthenticationInfo, secureServing *genericapiserver.SecureServingInfo) error {
	if o == nil {
		return nil
	}

	authenticatorConfig, err := o.ToAuthenticationConfig()
	if err != nil {
		return err
	}

	if authenticatorConfig.ClientCAContentProvider != nil {
		if err = authInfo.ApplyClientCert(authenticatorConfig.ClientCAContentProvider, secureServing); err != nil {
			return fmt.Errorf("unable to load client CA file: %v", err)
		}
	}
	if authenticatorConfig.RequestHeaderConfig != nil && authenticatorConfig.RequestHeaderConfig.CAContentProvider != nil {
		if err = authInfo.ApplyClientCert(authenticatorConfig.RequestHeaderConfig.CAContentProvider, secureServing); err != nil {
			return fmt.Errorf("unable to load client CA file: %v", err)
		}
	}

	authInfo.RequestHeaderConfig = authenticatorConfig.RequestHeaderConfig
	authInfo.APIAudiences = o.APIAudiences
	//if o.ServiceAccounts != nil && len(o.ServiceAccounts.Issuers) != 0 && len(o.APIAudiences) == 0 {
	//	authInfo.APIAudiences = authenticator.Audiences(o.ServiceAccounts.Issuers)
	//}

	//authenticatorConfig.ServiceAccountTokenGetter = serviceaccountcontroller.NewGetterFromClient(
	//	extclient,
	//	versionedInformer.Core().V1().Secrets().Lister(),
	//	versionedInformer.Core().V1().ServiceAccounts().Lister(),
	//	versionedInformer.Core().V1().Pods().Lister(),
	//)
	//authenticatorConfig.SecretsWriter = extclient.CoreV1()

	if authenticatorConfig.BootstrapToken {
		authenticatorConfig.BootstrapTokenAuthenticator = bootstrap.NewTokenAuthenticator(models.NewSecret())
	}

	//if egressSelector != nil {
	//	egressDialer, err := egressSelector.Lookup(egressselector.ControlPlane.AsNetworkContext())
	//	if err != nil {
	//		return err
	//	}
	//	authenticatorConfig.CustomDial = egressDialer
	//}

	authInfo.Authenticator, err = authenticatorConfig.New(ctx)
	if err != nil {
		return err
	}

	return nil
}

// ApplyAuthorization will conditionally modify the authentication options based on the authorization options
func (o *BuiltInAuthenticationOptions) ApplyAuthorization(authorization *BuiltInAuthorizationOptions) {
	if o == nil || authorization == nil || o.Anonymous == nil {
		return
	}

	// authorization ModeAlwaysAllow cannot be combined with AnonymousAuth.
	// in such a case the AnonymousAuth is stomped to false and you get a message
	if o.Anonymous.Allow && sets.NewString(authorization.Modes...).Has(authzmodes.ModeAlwaysAllow) {
		klog.Warningf("AnonymousAuth is not allowed with the AlwaysAllow authorizer. Resetting AnonymousAuth to false. You should use a different authorizer")
		o.Anonymous.Allow = false
	}
}

// #################### server/options/authentication.go

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

type RequestHeaderAuthenticationOptions struct {
	// ClientCAFile is the root certificate bundle to verify client certificates on incoming requests
	// before trusting usernames in headers.
	ClientCAFile string

	UsernameHeaders     []string
	GroupHeaders        []string
	ExtraHeaderPrefixes []string
	AllowedNames        []string
}

func (s *RequestHeaderAuthenticationOptions) Validate() []error {
	allErrors := []error{}

	if err := checkForWhiteSpaceOnly("requestheader-username-headers", s.UsernameHeaders...); err != nil {
		allErrors = append(allErrors, err)
	}
	if err := checkForWhiteSpaceOnly("requestheader-group-headers", s.GroupHeaders...); err != nil {
		allErrors = append(allErrors, err)
	}
	if err := checkForWhiteSpaceOnly("requestheader-extra-headers-prefix", s.ExtraHeaderPrefixes...); err != nil {
		allErrors = append(allErrors, err)
	}
	if err := checkForWhiteSpaceOnly("requestheader-allowed-names", s.AllowedNames...); err != nil {
		allErrors = append(allErrors, err)
	}

	if len(s.UsernameHeaders) > 0 && !caseInsensitiveHas(s.UsernameHeaders, "X-Remote-User") {
		klog.Warningf("--requestheader-username-headers is set without specifying the standard X-Remote-User header - API aggregation will not work")
	}
	if len(s.GroupHeaders) > 0 && !caseInsensitiveHas(s.GroupHeaders, "X-Remote-Group") {
		klog.Warningf("--requestheader-group-headers is set without specifying the standard X-Remote-Group header - API aggregation will not work")
	}
	if len(s.ExtraHeaderPrefixes) > 0 && !caseInsensitiveHas(s.ExtraHeaderPrefixes, "X-Remote-Extra-") {
		klog.Warningf("--requestheader-extra-headers-prefix is set without specifying the standard X-Remote-Extra- header prefix - API aggregation will not work")
	}

	return allErrors
}

func checkForWhiteSpaceOnly(flag string, headerNames ...string) error {
	for _, headerName := range headerNames {
		if len(strings.TrimSpace(headerName)) == 0 {
			return fmt.Errorf("empty value in %q", flag)
		}
	}

	return nil
}

func caseInsensitiveHas(headers []string, header string) bool {
	for _, h := range headers {
		if strings.EqualFold(h, header) {
			return true
		}
	}
	return false
}

// ToAuthenticationRequestHeaderConfig returns a RequestHeaderConfig config object for these options
// if necessary, nil otherwise.
func (s *RequestHeaderAuthenticationOptions) ToAuthenticationRequestHeaderConfig() (*authenticatorfactory.RequestHeaderConfig, error) {
	if len(s.ClientCAFile) == 0 {
		return nil, nil
	}

	caBundleProvider, err := dynamiccertificates.NewDynamicCAContentFromFile("request-header", s.ClientCAFile)
	if err != nil {
		return nil, err
	}

	return &authenticatorfactory.RequestHeaderConfig{
		UsernameHeaders:     headerrequest.StaticStringSlice(s.UsernameHeaders),
		GroupHeaders:        headerrequest.StaticStringSlice(s.GroupHeaders),
		ExtraHeaderPrefixes: headerrequest.StaticStringSlice(s.ExtraHeaderPrefixes),
		CAContentProvider:   caBundleProvider,
		AllowedClientNames:  headerrequest.StaticStringSlice(s.AllowedNames),
	}, nil
}

// ClientCertAuthenticationOptions provides different options for client cert auth. You should use `GetClientVerifyOptionFn` to
// get the verify options for your authenticator.
type ClientCertAuthenticationOptions struct {
	// ClientCA is the certificate bundle for all the signers that you'll recognize for incoming client certificates
	ClientCA string

	// CAContentProvider are the options for verifying incoming connections using mTLS and directly assigning to users.
	// Generally this is the CA bundle file used to authenticate client certificates
	// If non-nil, this takes priority over the ClientCA file.
	CAContentProvider dynamiccertificates.CAContentProvider
}

// GetClientVerifyOptionFn provides verify options for your authenticator while respecting the preferred order of verifiers.
func (s *ClientCertAuthenticationOptions) GetClientCAContentProvider() (dynamiccertificates.CAContentProvider, error) {
	if s.CAContentProvider != nil {
		return s.CAContentProvider, nil
	}

	if len(s.ClientCA) == 0 {
		return nil, nil
	}

	return dynamiccertificates.NewDynamicCAContentFromFile("client-ca-bundle", s.ClientCA)
}
