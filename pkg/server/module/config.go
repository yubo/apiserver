package server

import (
	"net"

	genericoptions "github.com/yubo/apiserver/pkg/server/options"
	"github.com/yubo/golib/util"
)

// cmd/kube-apiserver/app/options/options.go
// Config contains the Config while running a generic api server.
type Config struct {
	GenericServerRunOptions *genericoptions.ServerRunOptions                 `json:"generic"`
	SecureServing           *genericoptions.SecureServingOptionsWithLoopback `json:"secureServing"`
	InsecureServing         *genericoptions.DeprecatedInsecureServingOptions `json:"insecureServing"`
	Audit                   *genericoptions.AuditOptions                     `json:"audit"`
	Features                *genericoptions.FeatureOptions                   `json:"feature"`
	Authentication          *genericoptions.BuiltInAuthenticationOptions     `json:"authentication"`
	Authorization           *genericoptions.BuiltInAuthorizationOptions      `json:"authorization"`
	SelfSignedCerts         struct {
		AlternateDNS []string `json:"alternateDNS" description:"alternate DNS"`
		AlternateIPs []net.IP `json:"alternateIps" description:"alternate ips"`
	} `json:"SelfSignedCerts"`

	// TODO: move to authentication
	//ServiceAccountSigningKeyFile     string
	//ServiceAccountIssuer             serviceaccount.TokenGenerator
	//ServiceAccountTokenMaxExpiration time.Duration

	//EnableIndex bool `json:"enableIndex"`
	//EnableProfiling bool `json:"enableProfiling"`
	// EnableDiscovery bool
	// Requires generic profiling enabled
	//EnableContentionProfiling bool `json:"enableContentionProfiling"`
	//EnableMetrics             bool `json:"enableMetrics"`

	// swagger
	//EnableOpenAPI           bool                `json:"enableOpenAPI" flag:"openapi" description:"enable OpenAPI/Swagger"`
	//KeepAuthorizationHeader bool                `json:"keepAuthorizationHeader" description:"KeepAuthorizationHeader after a successful authentication"`

	//EnableExpvar  bool `json:"enableExpvar"`
	//EnableHealthz bool `json:"enableHealthz"`
}

// newConfig creates a new ServerRunOptions object with default parameters
func newConfig() *Config {
	return &Config{
		GenericServerRunOptions: genericoptions.NewServerRunOptions(),
		SecureServing:           genericoptions.NewSecureServingOptions().WithLoopback(),
		InsecureServing:         genericoptions.NewDeprecatedInsecureServingOptions(),
		Audit:                   genericoptions.NewAuditOptions(),
		Features:                genericoptions.NewFeatureOptions(),
		Authentication:          genericoptions.NewBuiltInAuthenticationOptions().WithAll(),
		Authorization:           genericoptions.NewBuiltInAuthorizationOptions(),
	}
}

func (p *Config) String() string {
	return util.Prettify(p)
}

// Validate will be called by config reader
func (p *Config) Validate() []error {
	var errs []error

	errs = append(errs, p.GenericServerRunOptions.Validate()...)
	errs = append(errs, p.SecureServing.Validate()...)
	errs = append(errs, p.InsecureServing.Validate()...)

	return errs
}
