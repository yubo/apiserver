module github.com/yubo/examples/otel-trace

go 1.16

require (
	github.com/emicklei/go-restful/v3 v3.7.4
	go.opentelemetry.io/contrib/instrumentation/github.com/emicklei/go-restful/otelrestful v0.29.0
	go.opentelemetry.io/otel v1.4.1
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.4.1
	go.opentelemetry.io/otel/sdk v1.4.1
	go.opentelemetry.io/otel/trace v1.4.1
)
