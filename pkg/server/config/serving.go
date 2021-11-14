package config

import (
	"context"
	"fmt"
	"net"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/yubo/apiserver/pkg/dynamiccertificates"
	"github.com/yubo/apiserver/pkg/server"
	cliflag "github.com/yubo/golib/cli/flag"
	"github.com/yubo/golib/configer"
	utilcert "github.com/yubo/golib/util/cert"
	utilerrors "github.com/yubo/golib/util/errors"
	"github.com/yubo/golib/util/keyutil"
	utilnet "github.com/yubo/golib/util/net"
	"k8s.io/klog/v2"
)

func NewSecureServingOptions() *SecureServingOptions {
	return &SecureServingOptions{
		BindAddress: net.ParseIP("0.0.0.0"),
		BindPort:    8443,
		Required:    true,
		ServerCert: GeneratableKeyCert{
			PairName:      "apiserver",
			CertDirectory: "/var/run/" + filepath.Base(os.Args[0]),
		},
	}
}

type SecureServingOptions struct {
	Enabled     *bool  `json:"enabled" flag:"secure-serving" default:"true" description:"enable the secure serving"`
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
	SNICertKeys cliflag.NamedCertKeyArray `json:"sniCertKeys" flag:"tls-sni-cert-key" description:"A pair of x509 certificate and private key file paths, optionally suffixed with a list of domain patterns which are fully qualified domain names, possibly with prefixed wildcard segments. The domain patterns also allow IP addresses, but IPs should only be used if the apiserver has visibility to the IP address requested by a client. If no domain patterns are provided, the names of the certificate are extracted. Non-wildcard matches trump over wildcard matches, explicit domain patterns trump over extracted names. For multiple key/certificate pairs, use the --tls-sni-cert-key multiple times. Examples: \"example.crt,example.key\" or \"foo.crt,foo.key:*.foo.com,foo.com\"."`
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

func (p *SecureServingOptions) Tags() map[string]*configer.FieldTag {
	tags := map[string]*configer.FieldTag{}

	{
		desc := "The port on which to serve HTTPS with authentication and authorization."
		if p.Required {
			desc += " It cannot be switched off with 0."
		} else {
			desc += " If 0, don't serve HTTPS at all."
		}

		tags["bindPort"] = &configer.FieldTag{Description: desc}
	}
	{
		tlsCipherPreferredValues := cliflag.PreferredTLSCipherNames()
		tlsCipherInsecureValues := cliflag.InsecureTLSCipherNames()
		desc := "Comma-separated list of cipher suites for the server. " +
			"If omitted, the default Go cipher suites will be used. \n" +
			"Preferred values: " + strings.Join(tlsCipherPreferredValues, ", ") + ". \n" +
			"Insecure values: " + strings.Join(tlsCipherInsecureValues, ", ") + "."
		tags["cipherSuites"] = &configer.FieldTag{Description: desc}
	}

	{
		tlsPossibleVersions := cliflag.TLSPossibleVersions()
		desc := "Minimum TLS version supported. " +
			"Possible values: " + strings.Join(tlsPossibleVersions, ", ")
		tags["minTLSVersion"] = &configer.FieldTag{Description: desc}
	}

	return tags
}

func (p *SecureServingOptions) DefaultExternalAddress() (net.IP, error) {
	if p.ExternalAddress != nil && !p.ExternalAddress.IsUnspecified() {
		return p.ExternalAddress, nil
	}
	return utilnet.ResolveBindAddress(p.BindAddress)
}

func (p *SecureServingOptions) Validate() error {
	if p == nil {
		return nil
	}

	errors := []error{}

	// SecureServingOptions
	if p.Required && p.BindPort < 1 || p.BindPort > 65535 {
		errors = append(errors, fmt.Errorf("--secure-port %v must be between 1 and 65535, inclusive. It cannot be turned off with 0", p.BindPort))
	} else if p.BindPort < 0 || p.BindPort > 65535 {
		errors = append(errors, fmt.Errorf("--secure-port %v must be between 0 and 65535, inclusive. 0 for turning off secure port", p.BindPort))
	}

	if (len(p.ServerCert.CertFile) != 0 || len(p.ServerCert.KeyFile) != 0) && p.ServerCert.GeneratedCert != nil {
		errors = append(errors, fmt.Errorf("cert/key file and in-memory certificate cannot both be set"))
	}

	return utilerrors.NewAggregate(errors)
}

// ApplyTo fills up serving information in the server configuration.
func (p *SecureServingOptions) ApplyTo(config **server.SecureServingInfo) error {
	if p == nil {
		return nil
	}
	if p.BindPort <= 0 && p.Listener == nil {
		return nil
	}

	if p.Listener == nil {
		var err error
		addr := net.JoinHostPort(p.BindAddress.String(), strconv.Itoa(p.BindPort))

		c := net.ListenConfig{}

		ctls := multipleControls{}
		if p.PermitPortSharing {
			ctls = append(ctls, permitPortReuse)
		}
		if p.PermitAddressSharing {
			ctls = append(ctls, permitAddressReuse)
		}
		if len(ctls) > 0 {
			c.Control = ctls.Control
		}

		p.Listener, p.BindPort, err = CreateListener(p.BindNetwork, addr, c)
		if err != nil {
			return fmt.Errorf("failed to create listener: %v", err)
		}
	} else {
		if _, ok := p.Listener.Addr().(*net.TCPAddr); !ok {
			return fmt.Errorf("failed to parse ip and port from listener")
		}
		p.BindPort = p.Listener.Addr().(*net.TCPAddr).Port
		p.BindAddress = p.Listener.Addr().(*net.TCPAddr).IP
	}

	*config = &server.SecureServingInfo{
		Listener:                     p.Listener,
		HTTP2MaxStreamsPerConnection: p.HTTP2MaxStreamsPerConnection,
	}
	c := *config

	serverCertFile, serverKeyFile := p.ServerCert.CertFile, p.ServerCert.KeyFile
	// load main cert
	if len(serverCertFile) != 0 || len(serverKeyFile) != 0 {
		var err error
		c.Cert, err = dynamiccertificates.NewDynamicServingContentFromFiles("serving-cert", serverCertFile, serverKeyFile)
		if err != nil {
			return err
		}
	} else if p.ServerCert.GeneratedCert != nil {
		c.Cert = p.ServerCert.GeneratedCert
	}

	if len(p.CipherSuites) != 0 {
		cipherSuites, err := cliflag.TLSCipherSuites(p.CipherSuites)
		if err != nil {
			return err
		}
		c.CipherSuites = cipherSuites
	}

	var err error
	c.MinTLSVersion, err = cliflag.TLSVersion(p.MinTLSVersion)
	if err != nil {
		return err
	}

	// load SNI certs
	certs := p.SNICertKeys.Certs()
	namedTLSCerts := make([]dynamiccertificates.SNICertKeyContentProvider, 0, len(certs))
	for _, nck := range certs {
		tlsCert, err := dynamiccertificates.NewDynamicSNIContentFromFiles("sni-serving-cert", nck.CertFile, nck.KeyFile, nck.Names...)
		namedTLSCerts = append(namedTLSCerts, tlsCert)
		if err != nil {
			return fmt.Errorf("failed to load SNI cert and key: %v", err)
		}
	}
	c.SNICerts = namedTLSCerts

	return nil
}

func (p *SecureServingOptions) MaybeDefaultWithSelfSignedCerts(publicAddress string, alternateDNS []string, alternateIPs []net.IP) error {
	if p == nil || (p.BindPort == 0 && p.Listener == nil) {
		return nil
	}

	keyCert := &p.ServerCert
	if len(keyCert.CertFile) != 0 || len(keyCert.KeyFile) != 0 {
		return nil
	}

	canReadCertAndKey := false
	if len(p.ServerCert.CertDirectory) > 0 {
		if len(p.ServerCert.PairName) == 0 {
			return fmt.Errorf("PairName is required if CertDirectory is set")
		}
		keyCert.CertFile = path.Join(p.ServerCert.CertDirectory, p.ServerCert.PairName+".crt")
		keyCert.KeyFile = path.Join(p.ServerCert.CertDirectory, p.ServerCert.PairName+".key")
		if canRead, err := utilcert.CanReadCertAndKey(keyCert.CertFile, keyCert.KeyFile); err != nil {
			return err
		} else {
			canReadCertAndKey = canRead
		}
	}

	if !canReadCertAndKey {
		// add either the bind address or localhost to the valid alternates
		if p.BindAddress.IsUnspecified() {
			alternateDNS = append(alternateDNS, "localhost")
		} else {
			alternateIPs = append(alternateIPs, p.BindAddress)
		}

		if cert, key, err := utilcert.GenerateSelfSignedCertKeyWithFixtures(publicAddress, alternateIPs, alternateDNS, p.ServerCert.FixtureDirectory); err != nil {
			return fmt.Errorf("unable to generate self signed cert: %v", err)
		} else if len(keyCert.CertFile) > 0 && len(keyCert.KeyFile) > 0 {
			if err := utilcert.WriteCert(keyCert.CertFile, cert); err != nil {
				return err
			}
			if err := keyutil.WriteKey(keyCert.KeyFile, key); err != nil {
				return err
			}
			klog.Infof("Generated self-signed cert (%s, %s)", keyCert.CertFile, keyCert.KeyFile)
		} else {
			p.ServerCert.GeneratedCert, err = dynamiccertificates.NewStaticCertKeyContent("Generated self signed cert", cert, key)
			if err != nil {
				return err
			}
			klog.Infof("Generated self-signed cert in-memory")
		}
	}

	return nil
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
