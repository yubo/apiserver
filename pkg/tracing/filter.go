package tracing

import (
	"net/http"

	"github.com/emicklei/go-restful/v3"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
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
	serviceName       string
	contextHeaderName string
	tp                oteltrace.TracerProvider
}

func WithServiceName(name string) Option {
	return func(o *options) {
		o.serviceName = name
	}
}
func WithContextHeaderName(name string) Option {
	return func(o *options) {
		o.contextHeaderName = name
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

	return func(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
		r := req.Request
		propagators := Propagators()
		ctx := propagators.Extract(r.Context(), propagation.HeaderCarrier(r.Header))
		route := req.SelectedRoutePath()
		spanName := route
		tracer := o.tp.Tracer(InstrumentationScope)

		ctx, otelSpan := tracer.Start(ctx, spanName,
			oteltrace.WithAttributes(semconv.NetAttributesFromHTTPRequest("tcp", r)...),
			oteltrace.WithAttributes(semconv.EndUserAttributesFromHTTPRequest(r)...),
			oteltrace.WithAttributes(semconv.HTTPServerAttributesFromHTTPRequest(o.serviceName, route, r)...),
			oteltrace.WithSpanKind(oteltrace.SpanKindServer),
		)
		defer otelSpan.End()

		// pass the span through the request context
		req.Request = req.Request.WithContext(ctx)

		//if o.contextHeaderName != "" {
		//	resp.AddHeader(o.contextHeaderName, otelSpan.SpanContext().TraceID().String())
		//}

		chain.ProcessFilter(req, resp)

		attrs := semconv.HTTPAttributesFromHTTPStatusCode(resp.StatusCode())
		spanStatus, spanMessage := semconv.SpanStatusFromHTTPStatusCode(resp.StatusCode())
		otelSpan.SetAttributes(attrs...)
		otelSpan.SetStatus(spanStatus, spanMessage)
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
		otelhttp.WithTracerProvider(o.tp),
	}

	handlerFunc := handler.ServeHTTP
	if o.contextHeaderName != "" {
		handlerFunc = func(w http.ResponseWriter, req *http.Request) {
			w.Header().Add(o.contextHeaderName, trace.SpanFromContext(req.Context()).SpanContext().TraceID().String())
			//props.Inject(req.Context(), propagation.HeaderCarrier(w.Header()))

			handler.ServeHTTP(w, req)
		}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// With Noop TracerProvider, the otelhttp still handles context propagation.
		// See https://github.com/open-telemetry/opentelemetry-go/tree/main/example/passthrough
		otelhttp.NewHandler(http.HandlerFunc(handlerFunc), o.serviceName, options...).ServeHTTP(w, req)
	})
}
