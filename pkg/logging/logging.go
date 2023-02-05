package logging

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/emicklei/go-restful/v3"
	"github.com/yubo/apiserver/pkg/proc/options"
	"github.com/yubo/apiserver/pkg/proc"
	"github.com/yubo/apiserver/pkg/request"
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
	oteltrace "go.opentelemetry.io/otel/trace"
	"k8s.io/klog/v2"
)

const (
	moduleName = "tracing"
	tracerName = "github.com/yubo/apiserver/pkg/tracing"
)

var (
	_module = &tracing{name: moduleName}
	hookOps = []v1.HookOps{{
		Hook:        _module.start,
		Owner:       moduleName,
		HookNum:     v1.ACTION_START,
		Priority:    v1.PRI_SYS_INIT,
		SubPriority: v1.PRI_M_TRACING,
	}, {
		Hook:        _module.stop,
		Owner:       moduleName,
		HookNum:     v1.ACTION_STOP,
		Priority:    v1.PRI_SYS_INIT,
		SubPriority: v1.PRI_M_TRACING,
	}}
)

type Config struct {
	RadioBased        float64           `json:"radioBased"`
	ServiceName       string            `json:"serviceName"`
	ContextHeaderName string            `json:"contextHeaderName"`
	Attributes        map[string]string `json:"attributes"`
	Debug             bool              `json:"debug"`
	Otel              *OtelConfig       `json:"otel"`
	Jaeger            *JaegerConfig     `json:"jaeger"`
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
		ServiceName:       filepath.Base(os.Args[0]),
		ContextHeaderName: "",
		RadioBased:        1.0,
	}
}

func (p Config) String() string {
	return util.Prettify(p)
}

func (p *Config) Validate() error {
	return nil
}

type tracing struct {
	name   string
	config *Config

	tracerProvider *sdktrace.TracerProvider
	propagators    propagation.TextMapPropagator
	tracer         oteltrace.Tracer
}

func (p *tracing) start(ctx context.Context) error {
	cf := newConfig()
	if err := proc.ReadConfig(p.name, cf); err != nil {
		return err
	}

	if cf.Otel == nil && cf.Jaeger == nil && !cf.Debug {
		return nil
	}
	p.config = cf

	if err := p.prepare(ctx); err != nil {
		return fmt.Errorf("failed to initialize jaeger: %s", err)

	}

	// add tracer filter
	if server, ok := options.APIServerFrom(ctx); ok {
		klog.V(3).Infof("added trace filter(%s, %s)", cf.ServiceName, cf.ContextHeaderName)
		server.Filter(p.filter())
	} else {
		klog.Warning("unable to get http server, traces filter not added")
	}

	//ops.SetContext(options.WithTracer(p.ctx, p.tracer))
	klog.Infof("tracing enabled %s", cf.ServiceName)

	return nil
}

func (p *tracing) stop(ctx context.Context) error {
	if p.tracerProvider != nil {
		return p.tracerProvider.Shutdown(ctx)
	}

	return nil
}

func (p *tracing) prepare(ctx context.Context) error {
	cf := p.config
	opts := []sdktrace.TracerProviderOption{
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(cf.RadioBased)),
	}

	if res, err := p.resource(ctx); err != nil {
		return fmt.Errorf("failed to create resource: %w", err)
	} else {
		opts = append(opts, sdktrace.WithResource(res))
	}

	if c := cf.Otel; c != nil {
		exporter, err := newOtelExporter(ctx, c)
		if err != nil {
			return err
		}
		opts = append(opts, sdktrace.WithSpanProcessor(
			sdktrace.NewBatchSpanProcessor(exporter),
		))
		klog.V(3).Infof("added otel exporter")
	}

	if c := cf.Jaeger; c != nil {
		exporter, err := newJaegerExporter(ctx, c)
		if err != nil {
			return err
		}
		opts = append(opts, sdktrace.WithSpanProcessor(
			sdktrace.NewBatchSpanProcessor(exporter),
		))
		klog.V(3).Infof("added jaeger exporter")
	}

	if cf.Debug {
		exporter, err := stdout.New(stdout.WithPrettyPrint())
		if err != nil {
			return err
		}
		opts = append(opts, sdktrace.WithBatcher(exporter))
		klog.V(3).Infof("added stdout exporter")
	}

	p.tracerProvider = sdktrace.NewTracerProvider(opts...)
	p.propagators = propagation.TraceContext{}
	p.tracer = p.tracerProvider.Tracer(
		tracerName,
		oteltrace.WithInstrumentationVersion("0.1"),
	)

	// set global propagator to tracecontext (the default is no-op).
	otel.SetTracerProvider(p.tracerProvider)
	otel.SetTextMapPropagator(p.propagators)

	return nil
}

func (p *tracing) resource(ctx context.Context) (*resource.Resource, error) {
	cf := p.config

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

// filter returns a restful.FilterFunction which will trace an incoming request.
//
// The service parameter should describe the name of the (virtual) server handling
// the request.  Options can be applied to configure the tracer and propagators
// used for this filter.
func (p *tracing) filter() restful.FilterFunction {
	cf := p.config

	return func(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
		r := req.Request
		ctx := p.propagators.Extract(r.Context(), propagation.HeaderCarrier(r.Header))
		route := req.SelectedRoutePath()
		spanName := route

		ctx, span := p.tracer.Start(ctx, spanName,
			oteltrace.WithAttributes(semconv.NetAttributesFromHTTPRequest("tcp", r)...),
			oteltrace.WithAttributes(semconv.EndUserAttributesFromHTTPRequest(r)...),
			oteltrace.WithAttributes(semconv.HTTPServerAttributesFromHTTPRequest(cf.ServiceName, route, r)...),
			oteltrace.WithSpanKind(oteltrace.SpanKindServer),
		)
		defer span.End()

		ctx = request.WithTracer(ctx, p.tracer)
		ctx = request.WithTraceID(ctx, span.SpanContext().TraceID().String())

		// pass the span through the request context
		req.Request = req.Request.WithContext(ctx)

		if cf.ContextHeaderName != "" {
			resp.AddHeader(cf.ContextHeaderName, span.SpanContext().TraceID().String())
		}

		chain.ProcessFilter(req, resp)

		attrs := semconv.HTTPAttributesFromHTTPStatusCode(resp.StatusCode())
		spanStatus, spanMessage := semconv.SpanStatusFromHTTPStatusCode(resp.StatusCode())
		span.SetAttributes(attrs...)
		span.SetStatus(spanStatus, spanMessage)
	}
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

func Register() {
	proc.RegisterHooks(hookOps)
	proc.AddConfig(moduleName, newConfig(), proc.WithConfigGroup("tracing"))
}
