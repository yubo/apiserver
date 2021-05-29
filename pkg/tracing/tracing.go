package tracing

import (
	"context"
	"fmt"
	"io"
	"runtime"

	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go"
	jaegercfg "github.com/uber/jaeger-client-go/config"
	"github.com/uber/jaeger-lib/metrics/prometheus"
	"github.com/yubo/apiserver/pkg/options"
	"github.com/yubo/golib/proc"
	"github.com/yubo/golib/util"
	"gopkg.in/yaml.v2"
	"k8s.io/klog/v2"
)

const (
	moduleName = "tracing"
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
	defConfig, _ = yaml.Marshal(&config{
		Jaeger: &jaegercfg.Configuration{
			Headers: &jaeger.HeadersConfig{
				TraceContextHeaderName: "request-id",
			},
			Sampler: &jaegercfg.SamplerConfig{
				Type:  "const",
				Param: 1.0,
			},
		},
		HttpBody:    true,
		HttpHeader:  false,
		RespTraceId: true,
	})
)

type config struct {
	Jaeger      *jaegercfg.Configuration
	HttpBody    bool
	HttpHeader  bool
	RespTraceId bool
	ServerName  string
}

func newConfig() *config {
	var cf config

	if jc, err := jaegercfg.FromEnv(); err != nil {
		cf.Jaeger = jc
	}

	if err := yaml.Unmarshal(defConfig, &cf); err != nil {
		panic(err)
	}

	return &cf
}

func (p config) String() string {
	return util.Prettify(p)
}

func (p *config) Validate() error {
	return nil
}

type tracing struct {
	config *config
	name   string
	tracer opentracing.Tracer
	closer io.Closer
	ctx    context.Context
	cancel context.CancelFunc
}

var (
	_module = &tracing{name: moduleName}
	hookOps = []proc.HookOps{{
		Hook:        _module.start,
		Owner:       moduleName,
		HookNum:     proc.ACTION_TEST,
		Priority:    proc.PRI_SYS_INIT,
		SubPriority: options.PRI_M_TRACING,
	}, {
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

func (p *tracing) init(ops *proc.HookOps) (err error) {
	ctx, configer := ops.ContextAndConfiger()
	p.ctx, p.cancel = context.WithCancel(ctx)

	cf := newConfig()
	if err := configer.ReadYaml(p.name, cf); err != nil {
		klog.ErrorS(err, "readYaml", "name", p.name)
		return nil
	}

	if cf.ServerName == "" {
		cf.ServerName = proc.NameFrom(ctx)
	}
	p.config = cf
	return
}

func (p *tracing) start(ops *proc.HookOps) (err error) {
	if err := p.init(ops); err != nil {
		return err
	}

	if p.config == nil {
		return nil
	}

	cf := p.config
	if p.tracer, p.closer, err = p.newTracer(cf.Jaeger); err != nil {
		return fmt.Errorf("failed to initialize jaeger: %s", err)
	}

	if p.tracer == nil {
		return nil
	}

	go func() {
		<-p.ctx.Done()
		p.closer.Close()
	}()

	// add tracer filter
	http, ok := options.GenericServerFrom(p.ctx)
	if !ok {
		return fmt.Errorf("unable to get http server")
	}
	http.Filter(WithTrace(cf.HttpHeader, cf.HttpBody, cf.RespTraceId))

	opentracing.SetGlobalTracer(p.tracer)

	//ops.SetContext(options.WithTracer(p.ctx, p.tracer))
	klog.Infof("tracing enabled %s", cf.ServerName)
	return nil
}

func (p *tracing) stop(ops *proc.HookOps) error {
	if p.cancel != nil {
		p.cancel()
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
func (p *tracing) newTracer(cf *jaegercfg.Configuration) (opentracing.Tracer, io.Closer, error) {
	if cf == nil || cf.Disabled {
		return nil, nil, nil
	}
	l := logger{}
	l.Infof("initializing Jaeger tracer")

	for k, v := range tracerSpanTags {
		cf.Tags = append(cf.Tags, opentracing.Tag{
			Key:   k,
			Value: v,
		})
	}

	return cf.NewTracer(jaegercfg.Logger(l),
		jaegercfg.Metrics(prometheus.New()))
}

type logger struct{}

func (l logger) Error(msg string) {
	klog.ErrorDepth(1, msg)
}

// Infof logs a message at info priority
func (l logger) Infof(msg string, args ...interface{}) {
	if klog.V(5).Enabled() {
		klog.InfoDepth(1, fmt.Sprintf(msg, args...))
	}
}

func Register() {
	proc.RegisterHooks(hookOps)
}
