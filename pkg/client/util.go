package client

import (
	"fmt"
	"net/url"
)

// DefaultServerURL converts a host, host:port, or URL string to the default base server API path
// to use with a Client at a given API version following the standard conventions for a
// Kubernetes API.
func DefaultServerURL(host string, defaultTLS bool) (*url.URL, error) {
	if host == "" {
		return nil, fmt.Errorf("host must be a URL or a host:port pair")
	}
	base := host
	hostURL, err := url.Parse(base)
	if err != nil || hostURL.Scheme == "" || hostURL.Host == "" {
		scheme := "http://"
		if defaultTLS {
			scheme = "https://"
		}
		hostURL, err = url.Parse(scheme + base)
		if err != nil {
			return nil, err
		}
		if hostURL.Path != "" && hostURL.Path != "/" {
			return nil, fmt.Errorf("host must be a URL or a host:port pair: %q", base)
		}
	}

	return hostURL, nil
}

// defaultServerUrlFor is shared between IsConfigTransportTLS and RESTClientFor. It
// requires Host and Version to be set prior to being called.
func defaultServerUrlFor(config *Config) (*url.URL, error) {
	// TODO: move the default to secure when the apiserver supports TLS by default
	// config.Insecure is taken to mean "I want HTTPS but don't bother checking the certs against a CA."
	hasCA := len(config.CAFile) != 0 || len(config.CAData) != 0
	hasCert := len(config.CertFile) != 0 || len(config.CertData) != 0
	defaultTLS := hasCA || hasCert || config.Insecure
	host := config.Host
	if host == "" {
		host = "localhost"
	}

	return DefaultServerURL(host, defaultTLS)
}
