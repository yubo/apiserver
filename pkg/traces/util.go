package traces

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"path/filepath"
	"time"

	"go.opentelemetry.io/otel/trace"
)

var (
	noopTracer = trace.NewNoopTracerProvider().Tracer("noop")
)

type HttpClientConfig struct {
	Insecure           bool   `yaml:"insecure"`
	Timeout            int    `yaml:"timeout"`
	InsecureSkipVerify bool   `yaml:"insecureSkipVerify"`
	IssuerCAPath       string `yaml:"IssuerCAPath"`
}

// NewHTTPClient returns a http.Client configured with the Agent options.
func NewHTTPClient(c *HttpClientConfig) (*http.Client, error) {
	if c.Insecure {
		return http.DefaultClient, nil
	}

	t := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 10 * time.Second,
			DualStack: true,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	if c.InsecureSkipVerify {
		t.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	} else {
		cert, err := getIssuerCACertFromPath(c.IssuerCAPath)
		if err != nil {
			return nil, err // the errors from this path have enough context already
		}

		if cert != nil {
			t.TLSClientConfig = &tls.Config{
				RootCAs: x509.NewCertPool(),
			}
			t.TLSClientConfig.RootCAs.AddCert(cert)
		}
	}

	return &http.Client{
		Timeout:   time.Duration(c.Timeout) * time.Second,
		Transport: t,
	}, nil
}

func getIssuerCACertFromPath(path string) (*x509.Certificate, error) {
	if path == "" {
		return nil, nil
	}

	rawCA, err := ioutil.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, fmt.Errorf("could not read the CA file %q: %w", path, err)
	}

	if len(rawCA) == 0 {
		return nil, fmt.Errorf("could not read the CA file %q: empty file", path)
	}

	block, _ := pem.Decode(rawCA)
	if block == nil {
		return nil, fmt.Errorf("cannot decode the contents of the CA file %q: %w", path, err)
	}

	return x509.ParseCertificate(block.Bytes)
}

func Start(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return TracerFrom(ctx).Start(ctx, spanName, opts...)
}

// {{{ context
type key int

const (
	tracerKey key = iota
)

func WithTracer(parent context.Context, tracer trace.Tracer) context.Context {
	return context.WithValue(parent, tracerKey, tracer)
}

func TracerFrom(ctx context.Context) trace.Tracer {
	if tracer, ok := ctx.Value(tracerKey).(trace.Tracer); ok {
		return tracer
	}
	return noopTracer
}

//}}}
