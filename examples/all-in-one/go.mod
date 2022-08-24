module examples/all-in-one

go 1.16

replace github.com/yubo/apiserver => ../..

require (
	github.com/go-openapi/spec v0.20.5
	github.com/spf13/cobra v1.4.0
	github.com/yubo/apiserver v0.1.0
	github.com/yubo/golib v0.0.3-0.20220824085241-a09a630bf3bf
	go.opentelemetry.io/otel v1.5.0
	go.opentelemetry.io/otel/trace v1.5.0
	k8s.io/klog/v2 v2.70.1
)
