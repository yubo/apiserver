package tracing

import (
	"net/http"

	"github.com/emicklei/go-restful/v3"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
)

var (
	DefaultOptions []Option
)

func SetOptions(opts ...Option) {
	DefaultOptions = append(DefaultOptions, opts...)
}

type Option func(*options)

func newOptions() *options {
	return &options{
		serviceName: "APIServer",
		tp:          NewNoopTracerProvider(),
	}
}

type options struct {
	serviceName string
	injectName  string
	tp          oteltrace.TracerProvider
}

func WithServiceName(name string) Option {
	return func(o *options) {
		o.serviceName = name
	}
}
func WithInjectName(name string) Option {
	return func(o *options) {
		o.injectName = name
	}
}
func WithTracerProvider(tp oteltrace.TracerProvider) Option {
	return func(o *options) {
		o.tp = tp
	}
}

// filter returns a restful.FilterFunction which will trace an incoming request.
//
// The service parameter should describe the name of the (virtual) server handling
// the request.  Options can be applied to configure the tracer and propagators
// used for this filter.
func RestfulFilter(opts ...Option) restful.FilterFunction {
	o := newOptions()
	for _, opt := range append(DefaultOptions, opts...) {
		opt(o)
	}

	props := Propagators()
	options := []otelhttp.Option{
		otelhttp.WithPropagators(props),
		otelhttp.WithPublicEndpoint(),
		otelhttp.WithTracerProvider(o.tp),
	}

	return func(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
		handler := func(w http.ResponseWriter, r *http.Request) {
			if o.injectName != "" {
				traceID := trace.SpanFromContext(r.Context()).SpanContext().TraceID()
				if traceID.IsValid() {
					w.Header().Add(o.injectName, traceID.String())
				}
			}

			req.Request = r
			chain.ProcessFilter(req, w.(*restful.Response))
		}
		otelhttp.NewHandler(http.HandlerFunc(handler), o.serviceName, options...).ServeHTTP(resp, req.Request)
	}
}

// WithTracing adds tracing to requests if the incoming request is sampled
func WithTracing(handler http.Handler, opts ...Option) http.Handler {
	o := newOptions()
	for _, opt := range append(DefaultOptions, opts...) {
		opt(o)
	}

	props := Propagators()
	options := []otelhttp.Option{
		otelhttp.WithPropagators(props),
		otelhttp.WithPublicEndpoint(),
		otelhttp.WithTracerProvider(o.tp),
	}

	h := handler
	if o.injectName != "" {
		h = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			traceID := trace.SpanFromContext(req.Context()).SpanContext().TraceID()
			if traceID.IsValid() {
				w.Header().Add(o.injectName, traceID.String())
			}

			handler.ServeHTTP(w, req)
		})
	}

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// With Noop TracerProvider, the otelhttp still handles context propagation.
		// See https://github.com/open-telemetry/opentelemetry-go/tree/main/example/passthrough
		otelhttp.NewHandler(h, o.serviceName, options...).ServeHTTP(w, req)
	})
}
