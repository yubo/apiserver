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

// Package options contains flags and options for initializing an apiserver
package apiserver

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/yubo/apiserver/pkg/apiserver"
	"github.com/yubo/apiserver/pkg/dynamiccertificates"
	apirequest "github.com/yubo/apiserver/pkg/request"
	"github.com/yubo/golib/api"
	cliflag "github.com/yubo/golib/cli/flag"
	"github.com/yubo/golib/configer"
	"github.com/yubo/golib/util"
	"github.com/yubo/golib/util/errors"
	utilerrors "github.com/yubo/golib/util/errors"
	"github.com/yubo/golib/util/sets"
	"k8s.io/klog/v2"
)

func NewConfig() *Config {
	return &Config{
		MaxRequestsInFlight:         400,
		MaxMutatingRequestsInFlight: 200,
		RequestTimeout:              api.NewDuration("60s"),
		LivezGracePeriod:            api.NewDuration("0s"),
		MinRequestTimeout:           api.NewDuration("1800s"),
		ShutdownDelayDuration:       api.NewDuration("0s"),
		JSONPatchMaxCopyBytes:       3 * 1024 * 1024,
		MaxRequestBodyBytes:         3 * 1024 * 1024,
		EnablePriorityAndFairness:   true,

		BindAddress: net.ParseIP("0.0.0.0"),
		BindPort:    8443,
		Required:    true,
		ServerCert: GeneratableKeyCert{
			PairName:      "apiserver",
			CertDirectory: "/var/run/" + filepath.Base(os.Args[0]),
		},
	}
}

// Config contains the Config while running a generic api server.
type Config struct {
	//Enabled bool `json:"enabled" default:"true" flag:"apiserver-enable" description:"api server enable"`

	// ServerRunOptions
	CorsAllowedOriginList       []string     `json:"corsAllowedOriginList"`
	HSTSDirectives              []string     `json:"hstsDirectives" flag:"strict-transport-security-directives" description:"List of directives for HSTS, comma separated. If this list is empty, then HSTS directives will not be added. Example: 'max-age=31536000,includeSubDomains,preload'"`
	ExternalHost                string       `json:"externalHost" flag:"external-hostname" description:"The hostname to use when generating externalized URLs for this master (e.g. Swagger API Docs or OpenID Discovery)."`
	MaxRequestsInFlight         int          `json:"maxRequestsInFlight" default:"400" flag:"max-requests-inflight" description:"The maximum number of non-mutating requests in flight at a given time. When the server exceeds this, it rejects requests. Zero for no limit."`
	MaxMutatingRequestsInFlight int          `json:"maxMutatingRequestsInFlight" default:"200" flag:"max-mutating-requests-inflight" description:"The maximum number of mutating requests in flight at a given time. When the server exceeds this, it rejects requests. Zero for no limit."`
	RequestTimeout              api.Duration `json:"requestTimeout" default:"60s" flag:"request-timeout" description:"An optional field indicating the duration a handler must keep a request open before timing it out. This is the default request timeout for requests but may be overridden by flags such as --min-request-timeout for specific types of requests."`
	GoawayChance                float64      `json:"goawayChance" flag:"goaway-chance" description:"To prevent HTTP/2 clients from getting stuck on a single apiserver, randomly close a connection (GOAWAY). The client's other in-flight requests won't be affected, and the client will reconnect, likely landing on a different apiserver after going through the load balancer again. This argument sets the fraction of requests that will be sent a GOAWAY. Clusters with single apiservers, or which don't use a load balancer, should NOT enable this. Min is 0 (off), Max is .02 (1/50 requests); .001 (1/1000) is a recommended starting point."`
	LivezGracePeriod            api.Duration `json:"livezGracePeriod" flag:"livez-grace-period" description:"This option represents the maximum amount of time it should take for apiserver to complete its startup sequence and become live. From apiserver's start time to when this amount of time has elapsed, /livez will assume that unfinished post-start hooks will complete successfully and therefore return true."`
	MinRequestTimeout           api.Duration `json:"minRequestTimeout" default:"1800s" flag:"min-request-timeout" description:"An optional field indicating the minimum number of seconds a handler must keep a request open before timing it out. Currently only honored by the watch request handler, which picks a randomized value above this number as the connection timeout, to spread out load."`
	ShutdownDelayDuration       api.Duration `json:"shutdownDelayDuration" flag:"shutdown-delay-duration" description:"Time to delay the termination. During that time the server keeps serving requests normally. The endpoints /healthz and /livez will return success, but /readyz immediately returns failure. Graceful termination starts after this delay has elapsed. This can be used to allow load balancer to stop sending traffic to this server."`
	// We intentionally did not add a flag for this option. Users of the
	// apiserver library can wire it to a flag.

	JSONPatchMaxCopyBytes int64 `json:"-"`
	// The limit on the request body size that would be accepted and
	// decoded in a write request. 0 means no limit.
	// We intentionally did not add a flag for this option. Users of the
	// apiserver library can wire it to a flag.
	MaxRequestBodyBytes       int64 `json:"maxRequestBodyBytes" flag:"max-resource-write-bytes" description:"The limit on the request body size that would be accepted and decoded in a write request."`
	EnablePriorityAndFairness bool  `json:"enablePriorityAndFairness" default:"true" flag:"enable-priority-and-fairness" description:"If true and the APIPriorityAndFairness feature gate is enabled, replace the max-in-flight handler with an enhanced one that queues and dispatches with priority and fairness"`

	// ################# SecureServingOptions
	BindAddress net.IP `json:"bindAddress" default:"0.0.0.0" flag:"bind-address" description:"The IP address on which to listen for the --secure-port port. The associated interface(s) must be reachable by the rest of the cluster, and by CLI/web clients. If blank or an unspecified address (0.0.0.0 or ::), all interfaces will be used."`
	BindPort    int    `json:"bindPort" default:"443" flag:"secure-port" description:"BindPort is ignored when Listener is set, will serve https even with 0."`
	BindNetwork string `json:"bindNework" default:"tcp" description:"BindNetwork is the type of network to bind to - accepts \"tcp\", \"tcp4\", and \"tcp6\"."`
	// Required set to true means that BindPort cannot be zero.
	Required bool `json:"-"`
	// ExternalAddress is the address advertised, even if BindAddress is a loopback. By default this
	// is set to BindAddress if the later no loopback, or to the first host interface address.
	ExternalAddress net.IP `json:"-"`

	// Listener is the secure server network listener.
	// either Listener or BindAddress/BindPort/BindNetwork is set,
	// if Listener is set, use it and omit BindAddress/BindPort/BindNetwork.
	Listener net.Listener `json:"-"`

	// ServerCert is the TLS cert info for serving secure traffic
	ServerCert GeneratableKeyCert `json:"serverCert"`

	// SNICertKeys are named CertKeys for serving secure traffic with SNI support.
	SNICertKeys []cliflag.NamedCertKey `json:"sniCertKeys" flag:"tls-sni-cert-key" description:"A pair of x509 certificate and private key file paths, optionally suffixed with a list of domain patterns which are fully qualified domain names, possibly with prefixed wildcard segments. The domain patterns also allow IP addresses, but IPs should only be used if the apiserver has visibility to the IP address requested by a client. If no domain patterns are provided, the names of the certificate are extracted. Non-wildcard matches trump over wildcard matches, explicit domain patterns trump over extracted names. For multiple key/certificate pairs, use the --tls-sni-cert-key multiple times. Examples: \"example.crt,example.key\" or \"foo.crt,foo.key:*.foo.com,foo.com\"."`
	// CipherSuites is the list of allowed cipher suites for the server.
	// Values are from tls package constants (https://golang.org/pkg/crypto/tls/#pkg-constants).
	CipherSuites []string `json:"cipherSuites" flag:"tls-cipher-suites" description:"-"`
	// MinTLSVersion is the minimum TLS version supported.
	// Values are from tls package constants (https://golang.org/pkg/crypto/tls/#pkg-constants).
	MinTLSVersion string `json:"minTLSVersion" flag:"tls-min-version" description:"-"`

	// HTTP2MaxStreamsPerConnection is the limit that the api server imposes on each client.
	// A value of zero means to use the default provided by golang's HTTP/2 support.
	HTTP2MaxStreamsPerConnection int `json:"http2MaxStreamsPerConnection" flag:"http2-max-streams-per-connection" description:"The limit that the server gives to clients for the maximum number of streams in an HTTP/2 connection. Zero means to use golang's default."`

	// PermitPortSharing controls if SO_REUSEPORT is used when binding the port, which allows
	// more than one instance to bind on the same address and port.
	PermitPortSharing bool `json:"permitPortSharing" flag:"permit-port-sharing" description:"If true, SO_REUSEPORT will be used when binding the port, which allows more than one instance to bind on the same address and port. [default=false]"`

	// PermitAddressSharing controls if SO_REUSEADDR is used when binding the port.
	PermitAddressSharing bool `json:"PermitAddressSharing" falg:"permit-address-sharing" description:"If true, SO_REUSEADDR will be used when binding the port. This allows binding to wildcard IPs like 0.0.0.0 and specific IPs in parallel, and it avoids waiting for the kernel to release sockets in TIME_WAIT state."`

	// other
	Host            string       `json:"host" default:"0.0.0.0" flag:"bind-host" description:"The IP address on which to listen for the --bind-port port. The associated interface(s) must be reachable by the rest of the cluster, and by CLI/web clients. If blank or an unspecified address (0.0.0.0 or ::), all interfaces will be used."` // BindAddress
	Port            int          `json:"port" default:"8080" flag:"bind-port" description:"The port on which to serve HTTPS with authentication and authorization. It cannot be switched off with 0."`                                                                                                                                         // BindPort is ignored when Listener is set, will serve https even with 0.
	Network         string       `json:"bindNetwork" flag:"cors-allowed-origins" description:"List of allowed origins for CORS, comma separated.  An allowed origin can be a regular expression to support subdomain matching. If this list is empty CORS will not be enabled."`                                                               // BindNetwork is the type of network to bind to - defaults to "tcp", accepts "tcp", "tcp4", and "tcp6".
	ShutdownTimeout api.Duration `json:"shutdownTimeout" default:"60s" description:"ShutdownTimeout is the timeout used for server shutdown. This specifies the timeout before server gracefully shutdown returns."`

	EnableIndex     bool `json:"enableIndex" default:"true"`
	EnableProfiling bool `json:"enableProfiling" default:"false"`
	// EnableDiscovery bool
	// Requires generic profiling enabled
	EnableContentionProfiling bool `json:"enableContentionProfiling" default:"true"`
	EnableMetrics             bool `json:"enableMetrics" default:"true"`
}

func (p *Config) Tags() map[string]*configer.TagOpts {
	tags := map[string]*configer.TagOpts{}

	{
		desc := "The port on which to serve HTTPS with authentication and authorization."
		if p.Required {
			desc += " It cannot be switched off with 0."
		} else {
			desc += " If 0, don't serve HTTPS at all."
		}

		tags["bindPort"] = &configer.TagOpts{Description: desc}
	}
	{
		tlsCipherPreferredValues := cliflag.PreferredTLSCipherNames()
		tlsCipherInsecureValues := cliflag.InsecureTLSCipherNames()
		desc := "Comma-separated list of cipher suites for the server. " +
			"If omitted, the default Go cipher suites will be used. \n" +
			"Preferred values: " + strings.Join(tlsCipherPreferredValues, ", ") + ". \n" +
			"Insecure values: " + strings.Join(tlsCipherInsecureValues, ", ") + "."
		tags["cipherSuites"] = &configer.TagOpts{Description: desc}
	}

	{
		tlsPossibleVersions := cliflag.TLSPossibleVersions()
		desc := "Minimum TLS version supported. " +
			"Possible values: " + strings.Join(tlsPossibleVersions, ", ")
		tags["minTLSVersion"] = &configer.TagOpts{Description: desc}
	}

	return tags
}

func (s *Config) String() string {
	return util.Prettify(s)
}

// Validate will be called by config reader
func (c *Config) Validate() error {
	if len(c.ExternalHost) == 0 {
		if hostname, err := os.Hostname(); err == nil {
			c.ExternalHost = hostname
		} else {
			return fmt.Errorf("error finding host name: %v", err)
		}
		klog.V(1).Infof("external host was not specified, using %v", c.ExternalHost)
	}

	errors := []error{}

	if c.LivezGracePeriod.Duration < 0 {
		errors = append(errors, fmt.Errorf("--livez-grace-period can not be a negative value"))
	}

	if c.MaxRequestsInFlight < 0 {
		errors = append(errors, fmt.Errorf("--max-requests-inflight can not be negative value"))
	}
	if c.MaxMutatingRequestsInFlight < 0 {
		errors = append(errors, fmt.Errorf("--max-mutating-requests-inflight can not be negative value"))
	}

	if c.RequestTimeout.Duration < 0 {
		errors = append(errors, fmt.Errorf("--request-timeout can not be negative value"))
	}

	if c.GoawayChance < 0 || c.GoawayChance > 0.02 {
		errors = append(errors, fmt.Errorf("--goaway-chance can not be less than 0 or greater than 0.02"))
	}

	if c.MinRequestTimeout.Duration < 0 {
		errors = append(errors, fmt.Errorf("--min-request-timeout can not be negative value"))
	}

	if c.ShutdownDelayDuration.Duration < 0 {
		errors = append(errors, fmt.Errorf("--shutdown-delay-duration can not be negative value"))
	}

	if c.MaxRequestBodyBytes < 0 {
		errors = append(errors, fmt.Errorf("--max-resource-write-bytes can not be negative value"))
	}

	if err := validateHSTSDirectives(c.HSTSDirectives); err != nil {
		errors = append(errors, err)
	}

	if c.Port < 1 || c.Port > 65535 {
		errors = append(errors, fmt.Errorf("--bind-port %v must be between 1 and 65535, inclusive. It cannot be turned off with 0", c.Port))
	}

	c.ExternalAddress = net.ParseIP(c.Host)

	// SecureServingOptions
	if c.Required && c.BindPort < 1 || c.BindPort > 65535 {
		errors = append(errors, fmt.Errorf("--secure-port %v must be between 1 and 65535, inclusive. It cannot be turned off with 0", c.BindPort))
	} else if c.BindPort < 0 || c.BindPort > 65535 {
		errors = append(errors, fmt.Errorf("--secure-port %v must be between 0 and 65535, inclusive. 0 for turning off secure port", c.BindPort))
	}

	if (len(c.ServerCert.CertFile) != 0 || len(c.ServerCert.KeyFile) != 0) && c.ServerCert.GeneratedCert != nil {
		errors = append(errors, fmt.Errorf("cert/key file and in-memory certificate cannot both be set"))
	}

	return utilerrors.NewAggregate(errors)
}

func NewRequestInfoResolver(c *Config) *apirequest.RequestInfoFactory {
	return &apirequest.RequestInfoFactory{
		APIPrefixes:          sets.NewString("api", "apis"),
		GrouplessAPIPrefixes: sets.NewString("api"),
	}
}

func validateHSTSDirectives(hstsDirectives []string) error {
	// HSTS Headers format: Strict-Transport-Security:max-age=expireTime [;includeSubDomains] [;preload]
	// See https://tools.ietf.org/html/rfc6797#section-6.1 for more information
	allErrors := []error{}
	for _, hstsDirective := range hstsDirectives {
		if len(strings.TrimSpace(hstsDirective)) == 0 {
			allErrors = append(allErrors, fmt.Errorf("empty value in strict-transport-security-directives"))
			continue
		}
		if hstsDirective != "includeSubDomains" && hstsDirective != "preload" {
			maxAgeDirective := strings.Split(hstsDirective, "=")
			if len(maxAgeDirective) != 2 || maxAgeDirective[0] != "max-age" {
				allErrors = append(allErrors, fmt.Errorf("--strict-transport-security-directives invalid, allowed values: max-age=expireTime, includeSubDomains, preload. see https://tools.ietf.org/html/rfc6797#section-6.1 for more information"))
			}
		}
	}
	return errors.NewAggregate(allErrors)
}

type GeneratableKeyCert struct {
	// CertFile is a file containing a PEM-encoded certificate, and possibly the complete certificate chain
	CertFile string `json:"certFile" flag:"tls-cert-file" description:"File containing the default x509 Certificate for HTTPS. (CA cert, if any, concatenated after server cert). If HTTPS serving is enabled, and --tls-cert-file and --tls-private-key-file are not provided, a self-signed certificate and key are generated for the public address and saved to the directory specified by --cert-dir."`
	// KeyFile is a file containing a PEM-encoded private key for the certificate specified by CertFile
	KeyFile string `json:"keyFile" flag:"tls-private-key-file" description:"File containing the default x509 private key matching --tls-cert-file."`

	// CertDirectory specifies a directory to write generated certificates to if CertFile/KeyFile aren't explicitly set.
	// PairName is used to determine the filenames within CertDirectory.
	// If CertDirectory and PairName are not set, an in-memory certificate will be generated.
	CertDirectory string `json:"certDir" flag:"cert-dir" description:"The directory where the TLS certs are located. If --tls-cert-file and --tls-private-key-file are provided, this flag will be ignored."`
	// PairName is the name which will be used with CertDirectory to make a cert and key filenames.
	// It becomes CertDirectory/PairName.crt and CertDirectory/PairName.key
	PairName string `json:"pairName" description:"It becomes CertDirectory/PairName.crt and CertDirectory/PairName.key"`

	// GeneratedCert holds an in-memory generated certificate if CertFile/KeyFile aren't explicitly set, and CertDirectory/PairName are not set.
	GeneratedCert dynamiccertificates.CertKeyContentProvider `json:"-"`

	// FixtureDirectory is a directory that contains test fixture used to avoid regeneration of certs during tests.
	// The format is:
	// <host>_<ip>-<ip>_<alternateDNS>-<alternateDNS>.crt
	// <host>_<ip>-<ip>_<alternateDNS>-<alternateDNS>.key
	FixtureDirectory string `json:"fixtureDirectory"`
}

func (s *Config) ServingInfo() (*apiserver.ServingInfo, error) {
	if s.BindPort <= 0 && s.Listener == nil {
		return nil, nil
	}

	if s.Listener == nil {
		var err error
		addr := net.JoinHostPort(s.BindAddress.String(), strconv.Itoa(s.BindPort))

		c := net.ListenConfig{}

		ctls := multipleControls{}
		if s.PermitPortSharing {
			ctls = append(ctls, permitPortReuse)
		}
		if s.PermitAddressSharing {
			ctls = append(ctls, permitAddressReuse)
		}
		if len(ctls) > 0 {
			c.Control = ctls.Control
		}

		s.Listener, s.BindPort, err = CreateListener(s.BindNetwork, addr, c)
		if err != nil {
			return nil, fmt.Errorf("failed to create listener: %v", err)
		}
	} else {
		if _, ok := s.Listener.Addr().(*net.TCPAddr); !ok {
			return nil, fmt.Errorf("failed to parse ip and port from listener")
		}
		s.BindPort = s.Listener.Addr().(*net.TCPAddr).Port
		s.BindAddress = s.Listener.Addr().(*net.TCPAddr).IP
	}

	c := &apiserver.ServingInfo{
		Listener:                     s.Listener,
		HTTP2MaxStreamsPerConnection: s.HTTP2MaxStreamsPerConnection,
	}

	serverCertFile, serverKeyFile := s.ServerCert.CertFile, s.ServerCert.KeyFile
	// load main cert
	if len(serverCertFile) != 0 || len(serverKeyFile) != 0 {
		var err error
		c.Cert, err = dynamiccertificates.NewDynamicServingContentFromFiles("serving-cert", serverCertFile, serverKeyFile)
		if err != nil {
			return nil, err
		}
	} else if s.ServerCert.GeneratedCert != nil {
		c.Cert = s.ServerCert.GeneratedCert
	}

	if len(s.CipherSuites) != 0 {
		cipherSuites, err := cliflag.TLSCipherSuites(s.CipherSuites)
		if err != nil {
			return nil, err
		}
		c.CipherSuites = cipherSuites
	}

	var err error
	c.MinTLSVersion, err = cliflag.TLSVersion(s.MinTLSVersion)
	if err != nil {
		return nil, err
	}

	// load SNI certs
	namedTLSCerts := make([]dynamiccertificates.SNICertKeyContentProvider, 0, len(s.SNICertKeys))
	for _, nck := range s.SNICertKeys {
		tlsCert, err := dynamiccertificates.NewDynamicSNIContentFromFiles("sni-serving-cert", nck.CertFile, nck.KeyFile, nck.Names...)
		namedTLSCerts = append(namedTLSCerts, tlsCert)
		if err != nil {
			return nil, fmt.Errorf("failed to load SNI cert and key: %v", err)
		}
	}
	c.SNICerts = namedTLSCerts

	return c, nil
}

func CreateListener(network, addr string, config net.ListenConfig) (net.Listener, int, error) {
	if len(network) == 0 {
		network = "tcp"
	}

	ln, err := config.Listen(context.TODO(), network, addr)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to listen on %v: %v", addr, err)
	}

	// get port
	tcpAddr, ok := ln.Addr().(*net.TCPAddr)
	if !ok {
		ln.Close()
		return nil, 0, fmt.Errorf("invalid listen address: %q", ln.Addr().String())
	}

	return ln, tcpAddr.Port, nil
}

type multipleControls []func(network, addr string, conn syscall.RawConn) error

func (mcs multipleControls) Control(network, addr string, conn syscall.RawConn) error {
	for _, c := range mcs {
		if err := c(network, addr, conn); err != nil {
			return err
		}
	}
	return nil
}
