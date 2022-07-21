package config

import (
	"net"

	"github.com/yubo/apiserver/pkg/rest"
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
		EnableOpenAPI:             true,
		EnableHealthz:             false,
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

	EnableIndex     bool `json:"enableIndex"`
	EnableProfiling bool `json:"enableProfiling"`
	// EnableDiscovery bool
	// Requires generic profiling enabled
	EnableContentionProfiling bool `json:"enableContentionProfiling"`
	EnableMetrics             bool `json:"enableMetrics"`

	// swagger
	//Swagger         goswagger.Config         `json:"swagger"`
	EnableOpenAPI bool `json:"enableOpenAPI" flag:"openapi" description:"enable OpenAPI"`
	//EnableSwagger   bool                `json:"enableSwagger" flag:"swagger" description:"enable swagger"`
	SecuritySchemes []rest.SchemeConfig `json:"securitySchemes"`

	EnableExpvar bool `json:"enableExpvar"`

	EnableHealthz bool `json:"enableHealthz"`
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
		EnableOpenAPI:          p.EnableOpenAPI,
		SecuritySchemes:        p.SecuritySchemes,
	}
}

func (p *Config) GetTags() map[string]*configer.FieldTag {
	tags := map[string]*configer.FieldTag{}

	for k, v := range p.GenericServerRunOptions.GetTags() {
		tags["generic."+k] = v
	}
	for k, v := range p.SecureServing.GetTags() {
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

	if len(p.SecuritySchemes) == 0 {
		p.SecuritySchemes = []rest.SchemeConfig{{
			Name:        "BearerToken",
			Type:        "apiKey",
			FieldName:   "authorization",
			ValueSource: "header",
		}}
	}

	return utilerrors.NewAggregate(errors)
}
