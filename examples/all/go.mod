module github.com/yubo/apiserver/examples/all

go 1.16

replace github.com/yubo/apiserver => ../..

require (
	github.com/opentracing/opentracing-go v1.2.0
	github.com/spf13/cobra v1.1.1
	github.com/yubo/apiserver v0.1.0
	github.com/yubo/golib v0.0.2-0.20220109150524-29ec97838a83
	google.golang.org/grpc v1.41.0
	google.golang.org/protobuf v1.27.1
	k8s.io/klog/v2 v2.9.0
)
