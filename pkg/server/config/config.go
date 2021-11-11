package config

import (
	"net"

	"github.com/yubo/apiserver/pkg/server"
	"github.com/yubo/golib/configer"
	"github.com/yubo/golib/scheme"
	"github.com/yubo/golib/util"
	utilerrors "github.com/yubo/golib/util/errors"
	"github.com/yubo/golib/util/sets"
)

func NewConfig() *Config {
	return &Config{
		GenericServerRunOptions: NewServerRunOptions(),
		SecureServing:           NewSecureServingOptions(),
		InsecureServing: &DeprecatedInsecureServingOptions{
			BindAddress: net.ParseIP("0.0.0.0"),
			BindPort:    8080,
			BindNetwork: "tcp",
		},
		EnableContentionProfiling: true,
		EnableExpvar:              false,
		EnableIndex:               true,
		EnableProfiling:           false,
		EnableMetrics:             true,
	}
}

// Config contains the Config while running a generic api server.
type Config struct {
	GenericServerRunOptions *ServerRunOptions                 `json:"generic"`
	SecureServing           *SecureServingOptions             `json:"secureServing"`
	InsecureServing         *DeprecatedInsecureServingOptions `json:"insecureServing"`

	AlternateDNS []string `json:"alternateDNS" flag:"alternate-dns" description:"alternate DNS"`

	// TODO: move to authentication
	//ServiceAccountSigningKeyFile     string
	//ServiceAccountIssuer             serviceaccount.TokenGenerator
	//ServiceAccountTokenMaxExpiration time.Duration

	//Host    string `json:"host" default:"0.0.0.0" flag:"bind-host" description:"The IP address on which to listen for the --bind-port port. The associated interface(s) must be reachable by the rest of the cluster, and by CLI/web clients. If blank or an unspecified address (0.0.0.0 or ::), all interfaces will be used."` // BindAddress
	//Port    int    `json:"port" default:"8080" flag:"bind-port" description:"The port on which to serve HTTPS with authentication and authorization. It cannot be switched off with 0."`                                                                                                                                         // BindPort is ignored when Listener is set, will serve https even with 0.
	//Network string `json:"bindNetwork" flag:"cors-allowed-origins" description:"List of allowed origins for CORS, comma separated.  An allowed origin can be a regular expression to support subdomain matching. If this list is empty CORS will not be enabled."`                                                               // BindNetwork is the type of network to bind to - defaults to "tcp", accepts "tcp", "tcp4", and "tcp6".

	EnableIndex     bool `json:"enableIndex"`
	EnableProfiling bool `json:"enableProfiling"`
	// EnableDiscovery bool
	// Requires generic profiling enabled
	EnableContentionProfiling bool `json:"enableContentionProfiling"`
	EnableMetrics             bool `json:"enableMetrics"`

	EnableExpvar bool `json:"enableExpvar"`
}

func (p *Config) NewServerConfig() *server.Config {
	return &server.Config{
		CorsAllowedOriginList:  p.GenericServerRunOptions.CorsAllowedOriginList,
		HSTSDirectives:         p.GenericServerRunOptions.HSTSDirectives,
		RequestTimeout:         p.GenericServerRunOptions.RequestTimeout.Duration,
		ShutdownTimeout:        p.GenericServerRunOptions.RequestTimeout.Duration,
		ShutdownDelayDuration:  p.GenericServerRunOptions.ShutdownDelayDuration.Duration,
		LegacyAPIGroupPrefixes: sets.NewString(server.DefaultLegacyAPIPrefix),
		Serializer:             scheme.Codecs.WithoutConversion(),
	}
}

func (p *Config) Tags() map[string]*configer.FieldTag {
	tags := map[string]*configer.FieldTag{}

	for k, v := range p.GenericServerRunOptions.Tags() {
		tags["generic."+k] = v
	}
	for k, v := range p.SecureServing.Tags() {
		tags["serving."+k] = v
	}

	return tags
}

func (p *Config) String() string {
	return util.Prettify(p)
}

// Validate will be called by config reader
func (p *Config) Validate() error {
	errors := []error{}

	if err := p.GenericServerRunOptions.Validate(); err != nil {
		errors = append(errors, err)
	}

	if p.SecureServing != nil && !*p.SecureServing.Enabled {
		p.SecureServing = nil
	}
	if err := p.SecureServing.Validate(); err != nil {
		errors = append(errors, err)
	}

	if p.InsecureServing != nil && !*p.InsecureServing.Enabled {
		p.InsecureServing = nil
	}
	if err := p.InsecureServing.Validate(); err != nil {
		errors = append(errors, err)
	}

	return utilerrors.NewAggregate(errors)
}
