module examples/all-in-one

go 1.16

replace github.com/yubo/apiserver => ../..

require (
	github.com/emicklei/go-restful/v3 v3.7.4
	github.com/go-openapi/spec v0.20.5
	github.com/spf13/cobra v1.4.0
	github.com/yubo/apiserver v0.1.0
	github.com/yubo/golib v0.0.3-0.20220825061925-f4cd420e40b5
	go.opentelemetry.io/otel v1.5.0
	go.opentelemetry.io/otel/trace v1.5.0
	k8s.io/klog/v2 v2.70.1
)
