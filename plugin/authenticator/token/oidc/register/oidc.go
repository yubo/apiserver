package register

import (
	"context"
	"fmt"

	"github.com/yubo/apiserver/pkg/authentication"
	"github.com/yubo/apiserver/pkg/authentication/authenticator"
	"github.com/yubo/apiserver/pkg/proc"
	"github.com/yubo/apiserver/plugin/authenticator/token/oidc"
	"k8s.io/klog/v2"
)

const (
	configPath       = "authentication.oidc"
	moduleName       = "authentication.oidc"
	noUsernamePrefix = "-"
)

type config struct {
	CAFile string `json:"caFile" flag:"oidc-ca-file" description:"If set, the OpenID server's certificate will be verified by one of the authorities in the oidc-ca-file, otherwise the host's root CA set will be used."`

	ClientID string `json:"clientID" flag:"oidc-client-id" description:"The client ID for the OpenID Connect client, must be set if oidc-issuer-url is set."`

	IssuerURL string `json:"issuerURL" flag:"oidc-issuer-url" description:"The URL of the OpenID issuer, only HTTPS scheme will be accepted. If set, it will be used to verify the OIDC JSON Web Token (JWT)."`

	UsernameClaim string `json:"usernameClaim" default:"sub" flag:"oidc-username-claim" description:"The OpenID claim to use as the user name. Note that claims other than the default ('sub') is not guaranteed to be unique and immutable. This flag is experimental, please see the authentication documentation for further details."`

	UsernamePrefix string `json:"usernamePrefix" flag:"oidc-username-prefix" description:"If provided, all usernames will be prefixed with this value. If not provided, username claims other than 'email' are prefixed by the issuer URL to avoid clashes. To skip any prefixing, provide the value '-'."`

	GroupsClaim string `json:"groupsClaim" flag:"oidc-groups-claim" description:"If provided, the name of a custom OpenID Connect claim for specifying user groups. The claim value is expected to be a string or array of strings. This flag is experimental, please see the authentication documentation for further details."`

	GroupsPrefix string `json:"groupsPrefix" flag:"oidc-groups-prefix" description:"If provided, all groups will be prefixed with this value to prevent conflicts with other authentication strategies."`

	SigningAlgs []string `json:"signingAlgs" default:"RS256" flag:"oidc-signing-algs" description:"Comma-separated list of allowed JOSE asymmetric signing algorithms. JWTs with a 'alg' header value not in this list will be rejected. Values are defined by RFC 7518 https://tools.ietf.org/html/rfc7518#section-3.1."`

	// RequiredClaims, if specified, causes the OIDCAuthenticator to verify that all the
	// required claims key value pairs are present in the ID Token.
	RequiredClaims map[string]string `json:"requiredClaims" flag:"oidc-required-claim" description:"A key=value pair that describes a required claim in the ID Token. If set, the claim is verified to be present in the ID Token with a matching value. Repeat this flag to specify multiple claims."`
}

func (o *config) Validate() error {
	if (len(o.IssuerURL) > 0) != (len(o.ClientID) > 0) {
		return fmt.Errorf("oidc-issuer-url and oidc-client-id should be specified together")
	}

	if o.UsernamePrefix == "" && o.UsernameClaim != "email" {
		// Old behavior. If a usernamePrefix isn't provided, prefix all claims other than "email"
		// with the issuerURL.
		//
		// See https://github.com/kubernetes/kubernetes/issues/31380
		o.UsernamePrefix = o.IssuerURL + "#"
	}

	if o.UsernamePrefix == noUsernamePrefix {
		// Special value indicating usernames shouldn't be prefixed.
		o.UsernamePrefix = ""
	}

	return nil
}

func newConfig() *config {
	return &config{}
}

func factory(ctx context.Context) (authenticator.Token, error) {
	cf := newConfig()
	if err := proc.ReadConfig(configPath, cf); err != nil {
		return nil, err
	}

	if len(cf.IssuerURL) == 0 {
		klog.V(5).Infof("%s.issuerURL is not set, skip", configPath)
		return nil, nil
	}
	klog.V(5).InfoS("authmodule init", "name", moduleName, "IssuerURL", cf.IssuerURL)

	return oidc.New(oidc.Options{
		IssuerURL:            cf.IssuerURL,
		ClientID:             cf.ClientID,
		CAFile:               cf.CAFile,
		UsernameClaim:        cf.UsernameClaim,
		UsernamePrefix:       cf.UsernamePrefix,
		GroupsClaim:          cf.GroupsClaim,
		GroupsPrefix:         cf.GroupsPrefix,
		SupportedSigningAlgs: cf.SigningAlgs,
		RequiredClaims:       cf.RequiredClaims,
	})
}

func init() {
	authentication.RegisterTokenAuthn(factory)
	proc.AddConfig(configPath, newConfig(), proc.WithConfigGroup("authentication"))
}
