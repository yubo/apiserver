package apiserver

import (
	"net/http"

	"github.com/emicklei/go-restful"
	"github.com/yubo/apiserver/pkg/dynamiccertificates"
)

type APIServer interface {
	Handle(string, http.Handler)
	HandleFunc(string, func(http.ResponseWriter, *http.Request))
	UnlistedHandle(string, http.Handler)
	UnlistedHandleFunc(string, func(http.ResponseWriter, *http.Request))
	Add(*restful.WebService) *restful.Container
	Filter(restful.FilterFunction)
	Address() string
	ServingInfo() *ServingInfo
}

type ServingInfo struct {
	// Cert is the main server cert which is used if SNI does not match. Cert must be non-nil and is
	// allowed to be in SNICerts.
	Cert dynamiccertificates.CertKeyContentProvider

	// SNICerts are the TLS certificates used for SNI.
	SNICerts []dynamiccertificates.SNICertKeyContentProvider

	// ClientCA is the certificate bundle for all the signers that you'll recognize for incoming client certificates
	ClientCA dynamiccertificates.CAContentProvider

	// MinTLSVersion optionally overrides the minimum TLS version supported.
	// Values are from tls package constants (https://golang.org/pkg/crypto/tls/#pkg-constants).
	MinTLSVersion uint16

	// CipherSuites optionally overrides the list of allowed cipher suites for the server.
	// Values are from tls package constants (https://golang.org/pkg/crypto/tls/#pkg-constants).
	CipherSuites []uint16

	// HTTP2MaxStreamsPerConnection is the limit that the api server imposes on each client.
	// A value of zero means to use the default provided by golang's HTTP/2 support.
	HTTP2MaxStreamsPerConnection int

	// DisableHTTP2 indicates that http2 should not be enabled.
	DisableHTTP2 bool
}

func (p *ServingInfo) ApplyClientCert(clientCA dynamiccertificates.CAContentProvider) error {
	if p == nil || clientCA == nil {
		return nil
	}

	if p.ClientCA == nil {
		p.ClientCA = clientCA
		return nil
	}

	p.ClientCA = dynamiccertificates.NewUnionCAContentProvider(p.ClientCA, clientCA)

	return nil
}
