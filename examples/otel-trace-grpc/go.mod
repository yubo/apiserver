module examples/otel-trace-grpc

go 1.16

replace github.com/yubo/apiserver => ../..

replace github.com/yubo/golib => ../../../golib

replace github.com/yubo/client-go => ../../../client-go

require (
	github.com/yubo/apiserver v0.0.0-00010101000000-000000000000
	github.com/yubo/golib v0.0.3-0.20220902030005-7f15ca001a44
	go.opentelemetry.io/otel v1.13.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.12.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.5.0
	go.opentelemetry.io/otel/sdk v1.12.0
	go.opentelemetry.io/otel/trace v1.13.0
	google.golang.org/grpc v1.52.3
	google.golang.org/protobuf v1.28.1
)
