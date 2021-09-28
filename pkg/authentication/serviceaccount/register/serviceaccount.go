package register

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/yubo/apiserver/pkg/options"
	"github.com/yubo/golib/proc"
	utilerrors "github.com/yubo/golib/util/errors"
)

const (
	moduleName       = "authentication.serviceAccount"
	noUsernamePrefix = "-"
)

var (
	_auth   = &authModule{name: moduleName}
	hookOps = []proc.HookOps{{
		Hook:        _auth.init,
		Owner:       moduleName,
		HookNum:     proc.ACTION_START,
		Priority:    proc.PRI_SYS_INIT,
		SubPriority: options.PRI_M_AUTHN - 1,
	}}
	_config *config
)

type config struct {
	KeyFiles []string `json:"keyFiles" flags:"service-account-key-file" description:"File containing PEM-encoded x509 RSA or ECDSA private or public keys, used to verify ServiceAccount tokens. The specified file can contain multiple keys, and the flag can be specified multiple times with different files. If unspecified, --tls-private-key-file is used. Must be specified when --service-account-signing-key is provided"`

	Lookup bool `json:"lookup" default:"true" flags:"service-account-lookup" description:"If true, validate ServiceAccount tokens exist in etcd as part of authentication."`

	Issuer string `json:"issuer" flags:"service-account-issuer" description:"Identifier of the service account token issuer. The issuer will assert this identifier in \"iss\" claim of issued tokens. This value is a string or URI. If this option is not a valid URI per the OpenID Discovery 1.0 spec, the ServiceAccountIssuerDiscovery feature will remain disabled, even if the feature gate is set to true. It is highly recommended that this value comply with the OpenID spec: https://openid.net/specs/openid-connect-discovery-1_0.html. In practice, this means that service-account-issuer must be an https URL. It is also highly recommended that this URL be capable of serving OpenID discovery documents at {service-account-issuer}/.well-known/openid-configuration."`

	JWKSURI string `json:"jwksUri" flags:"service-account-jwks-uri" description:"Overrides the URI for the JSON Web Key Set in the discovery doc served at /.well-known/openid-configuration. This flag is useful if the discovery doc and key set are served to relying parties from a URL other than the API server's external (as auto-detected or overridden with external-hostname). Only valid if the ServiceAccountIssuerDiscovery feature gate is enabled."`

	MaxExpiration int `json:"maxExpiration" flags:"service-account-max-token-expiration" description:"The maximum validity duration of a token created by the service account token issuer. If an otherwise valid TokenRequest with a validity duration larger than this value is requested, a token will be issued with a validity duration of this value."`

	ExtendExpiration bool `json:"extendExpiration" default:"true" flags:"service-account-extend-token-expiration" description:"Turns on projected service account expiration extension during token generation, which helps safe transition from legacy token to bound service account token feature. If this flag is enabled, admission injected tokens would be extended up to 1 year to prevent unexpected failure during transition, ignoring value of service-account-max-token-expiration."`

	maxExpiration time.Duration
}

func (o *config) Validate() error {
	allErrors := []error{}

	if len(o.Issuer) == 0 {
		return nil
	}

	o.maxExpiration = time.Duration(o.MaxExpiration) * time.Second

	if len(o.Issuer) > 0 && strings.Contains(o.Issuer, ":") {
		if _, err := url.Parse(o.Issuer); err != nil {
			allErrors = append(allErrors, fmt.Errorf("service-account-issuer contained a ':' but was not a valid URL: %v", err))
		}
	}

	if len(o.Issuer) == 0 {
		allErrors = append(allErrors, errors.New("service-account-issuer is a required flag"))
	}
	if len(o.KeyFiles) == 0 {
		allErrors = append(allErrors, errors.New("service-account-key-file is a required flag"))
	}

	// Validate the JWKS URI when it is explicitly set.
	// When unset, it is later derived from ExternalHost.
	if o.JWKSURI != "" {
		if u, err := url.Parse(o.JWKSURI); err != nil {
			allErrors = append(allErrors, fmt.Errorf("service-account-jwks-uri must be a valid URL: %v", err))
		} else if u.Scheme != "https" {
			allErrors = append(allErrors, fmt.Errorf("service-account-jwks-uri requires https scheme, parsed as: %v", u.String()))
		}
	}

	return utilerrors.NewAggregate(allErrors)
}

type authModule struct {
	name   string
	config *config
}

func newConfig() *config {
	return &config{}
}

func (p *authModule) init(ctx context.Context) error {
	c := proc.ConfigerMustFrom(ctx)

	cf := newConfig()
	if err := c.Read(moduleName, cf); err != nil {
		return err
	}
	p.config = cf

	// TODO:
	return nil
}

func init() {
	proc.RegisterHooks(hookOps)
	proc.RegisterFlags(moduleName, "authentication", newConfig())
}
