package traces

import (
	"context"
	"fmt"
	"runtime"

	jaegercfg "github.com/uber/jaeger-client-go/config"
	"github.com/yubo/apiserver/pkg/options"
	"github.com/yubo/golib/configer"
	"github.com/yubo/golib/proc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc"
	"k8s.io/klog/v2"
)

const (
	moduleName = "traces"
)

var (
	tracerSpanTags = map[string]string{
		"build.revision":   options.Revision,
		"build.version":    options.Branch,
		"build.branch":     options.Version,
		"build.date":       options.BuildDate,
		"build.time_unix":  options.BuildTimeUnix,
		"build.go_version": runtime.Version(),
	}
)

type traces struct {
	config *Config
	name   string
	ctx    context.Context
	cancel context.CancelFunc

	provider *sdktrace.TracerProvider
}

var (
	_module = &traces{name: moduleName}
	hookOps = []proc.HookOps{{
		Hook:        _module.start,
		Owner:       moduleName,
		HookNum:     proc.ACTION_START,
		Priority:    proc.PRI_SYS_INIT,
		SubPriority: options.PRI_M_TRACING,
	}, {
		Hook:        _module.stop,
		Owner:       moduleName,
		HookNum:     proc.ACTION_STOP,
		Priority:    proc.PRI_SYS_INIT,
		SubPriority: options.PRI_M_TRACING,
	}}
)

func (p *traces) init(ctx context.Context) (err error) {
	c := configer.ConfigerMustFrom(ctx)
	p.ctx, p.cancel = context.WithCancel(ctx)

	cf := newConfig()
	if err := c.Read(p.name, cf); err != nil {
		klog.ErrorS(err, "readYaml", "name", p.name)
		return nil
	}

	p.config = cf
	return
}

func (p *traces) start(ctx context.Context) (err error) {
	if err := p.init(ctx); err != nil {
		return err
	}

	if p.config == nil {
		return nil
	}

	if p.provider, err = p.newTracer(ctx); err != nil {
		return fmt.Errorf("failed to initialize jaeger: %s", err)
	}

	// add tracer filter
	server, ok := options.APIServerFrom(p.ctx)
	if !ok {
		return fmt.Errorf("unable to get http server")
	}
	server.Filter(OTelFilter(p.config.ServiceName))

	//ops.SetContext(options.WithTracer(p.ctx, p.tracer))
	klog.Infof("tracing enabled %s", p.config.ServiceName)
	return nil
}

func (p *traces) stop(ctx context.Context) error {
	if p.provider != nil {
		p.provider.Shutdown(ctx)
	}
	return nil
}

// TracingConfiguration configures an opentracing backend for m3query to use. Currently only jaeger is supported.
// Tracing is disabled if no backend is specified.
type TracingConfiguration struct {
	Jaeger jaegercfg.Configuration
}

// NewTracer returns a tracer configured with the configuration provided by this struct. The tracer's concrete
// type is determined by cfg.Backend. Currently only `"jaeger"` is supported. `""` implies
// disabled (NoopTracer).
func (p *traces) newTracer(ctx context.Context) (*sdktrace.TracerProvider, error) {
	opts := []sdktrace.TracerProviderOption{
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	}

	if tags := p.config.Attributes; tags != nil {
		attrs := make([]attribute.KeyValue, 0, len(tags))
		for k, v := range tags {
			attrs = append(attrs, attribute.String(k, v))
		}
		res, err := resource.New(ctx, resource.WithAttributes(attrs...))
		if err != nil {
			return nil, fmt.Errorf("failed to create resource: %w", err)
		}
		opts = append(opts, sdktrace.WithResource(res))
	}

	if c := p.config.OTel; c != nil {
		driverOpts := []otlptracegrpc.Option{
			otlptracegrpc.WithEndpoint(c.Endpoint),
			otlptracegrpc.WithDialOption(grpc.WithBlock()),
		}
		if c.Insecure {
			driverOpts = append(driverOpts, otlptracegrpc.WithInsecure())
		}
		driver := otlptracegrpc.NewClient(driverOpts...)
		exporter, err := otlptrace.New(ctx, driver)
		if err != nil {
			return nil, fmt.Errorf("failed to trace exporter: %w", err)
		}
		opts = append(opts, sdktrace.WithSpanProcessor(
			sdktrace.NewBatchSpanProcessor(exporter),
		))
	}

	if c := p.config.Jaeger; c != nil {
		fmt.Printf("%v\n", c)
	}

	provider := sdktrace.NewTracerProvider(opts...)
	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	return provider, nil
}

func Register() {
	proc.RegisterHooks(hookOps)
	proc.RegisterFlags(moduleName, "traces", newConfig())
}
