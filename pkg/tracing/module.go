package tracing

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/yubo/apiserver/pkg/proc"
	v1 "github.com/yubo/apiserver/pkg/proc/api/v1"
	"github.com/yubo/golib/util"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	stdout "go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"k8s.io/klog/v2"
)

func Register(opts ...proc.ModuleOption) {
	o := &proc.ModuleOptions{
		Proc: proc.DefaultProcess,
	}
	for _, v := range opts {
		v(o)
	}

	module := &tracingT{name: "tracing"}
	hookOps := []v1.HookOps{{
		Hook:        module.start,
		Owner:       module.name,
		HookNum:     v1.ACTION_START,
		Priority:    v1.PRI_SYS_INIT,
		SubPriority: v1.PRI_M_TRACING,
	}, {
		Hook:        module.stop,
		Owner:       module.name,
		HookNum:     v1.ACTION_STOP,
		Priority:    v1.PRI_SYS_INIT,
		SubPriority: v1.PRI_M_TRACING,
	}}

	o.Proc.RegisterHooks(hookOps)
	o.Proc.AddConfig(module.name, newConfig(), proc.WithConfigGroup(module.name))
}

type Config struct {
	RadioBased  float64           `json:"radioBased"`
	ServiceName string            `json:"serviceName"`
	InjectName  string            `json:"injectName"`
	Attributes  map[string]string `json:"attributes"`
	Debug       bool              `json:"debug"`
	Otel        *OtelConfig       `json:"otel"`
	Jaeger      *JaegerConfig     `json:"jaeger"`
}

type OtelConfig struct {
	Endpoint string `json:"endpoint"`
	Insecure bool   `json:"insecure"`
}

type JaegerConfig struct {
	Endpoint string `json:"endpoint"`
	Username string `json:"username"`
	Password string `json:"password"`
	HttpClientConfig
}

func newConfig() *Config {
	return &Config{
		ServiceName: filepath.Base(os.Args[0]),
		InjectName:  "",
		RadioBased:  1.0,
	}
}

func (p Config) String() string {
	return util.Prettify(p)
}

func (p *Config) Validate() error {
	return nil
}

type tracingT struct {
	name        string
	tp          TracerProvider
	propagators propagation.TextMapPropagator
}

func (p *tracingT) start(ctx context.Context) error {
	cf := newConfig()
	if err := proc.ReadConfig(p.name, cf); err != nil {
		return err
	}

	//if cf.Otel == nil && cf.Jaeger == nil && !cf.Debug {
	//	return nil
	//}

	tp, err := p.initProvider(ctx, cf)
	if err != nil {
		return fmt.Errorf("failed to initialize jaeger: %s", err)
	}
	props := Propagators()

	// set global propagator to tracecontext (the default is no-op).
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(props)

	// set defaltOptions for tracing.filter
	SetOptions(
		WithTracerProvider(tp),
		WithServiceName(cf.ServiceName),
		WithInjectName(cf.InjectName),
	)

	//ops.SetContext(options.WithTracer(p.ctx, p.tracer))
	klog.Infof("tracing enabled %s", cf.ServiceName)

	p.tp = tp
	return nil
}

func (p *tracingT) stop(ctx context.Context) error {
	if p.tp != nil {
		return p.tp.Shutdown(ctx)
	}

	return nil
}

func (p *tracingT) initProvider(ctx context.Context, cf *Config) (*sdktrace.TracerProvider, error) {
	opts := []sdktrace.TracerProviderOption{
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(cf.RadioBased)),
	}

	if res, err := p.resource(ctx, cf); err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	} else {
		opts = append(opts, sdktrace.WithResource(res))
	}

	if c := cf.Otel; c != nil {
		exporter, err := newOtelExporter(ctx, c)
		if err != nil {
			return nil, err
		}
		opts = append(opts, sdktrace.WithSpanProcessor(
			sdktrace.NewBatchSpanProcessor(exporter),
		))
		klog.V(3).Infof("added otel exporter")
	}

	if c := cf.Jaeger; c != nil {
		exporter, err := newJaegerExporter(ctx, c)
		if err != nil {
			return nil, err
		}
		opts = append(opts, sdktrace.WithSpanProcessor(
			sdktrace.NewBatchSpanProcessor(exporter),
		))
		klog.V(3).Infof("added jaeger exporter")
	}

	if cf.Debug {
		exporter, err := stdout.New(stdout.WithPrettyPrint())
		if err != nil {
			return nil, err
		}
		opts = append(opts, sdktrace.WithBatcher(exporter))
		klog.V(3).Infof("added stdout exporter")
	}

	return sdktrace.NewTracerProvider(opts...), nil
}

func (p *tracingT) resource(ctx context.Context, cf *Config) (*resource.Resource, error) {
	opts := []resource.Option{
		resource.WithProcess(),
		resource.WithHost(),
		resource.WithAttributes(
			// the service name used to display traces in backends
			semconv.ServiceNameKey.String(cf.ServiceName),
		),
	}

	if len(cf.Attributes) > 0 {
		attrs := make([]attribute.KeyValue, 0, len(cf.Attributes))
		for k, v := range cf.Attributes {
			attrs = append(attrs, attribute.String(k, v))
		}
		opts = append(opts, resource.WithAttributes(attrs...))
	}

	return resource.New(ctx, opts...)
}

func newOtelExporter(ctx context.Context, c *OtelConfig) (*otlptrace.Exporter, error) {
	driverOpts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(c.Endpoint),
		//otlptracegrpc.WithDialOption(grpc.WithBlock()),
	}
	if c.Insecure {
		driverOpts = append(driverOpts, otlptracegrpc.WithInsecure())
	}
	return otlptrace.New(ctx, otlptracegrpc.NewClient(driverOpts...))
}

func newJaegerExporter(ctx context.Context, c *JaegerConfig) (*jaeger.Exporter, error) {
	// Create the Jaeger exporter
	opts := []jaeger.CollectorEndpointOption{}
	if len(c.Endpoint) > 0 {
		opts = append(opts, jaeger.WithEndpoint(c.Endpoint))
	}
	if len(c.Username) > 0 {
		opts = append(opts, jaeger.WithUsername(c.Username))
	}
	if len(c.Password) > 0 {
		opts = append(opts, jaeger.WithPassword(c.Password))
	}
	if client, err := NewHTTPClient(&c.HttpClientConfig); err != nil {
		return nil, err
	} else {
		opts = append(opts, jaeger.WithHTTPClient(client))
	}

	return jaeger.New(jaeger.WithCollectorEndpoint(opts...))
}

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
