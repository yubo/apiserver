module examples/websocket-watch

go 1.16

replace github.com/yubo/apiserver => ../..

require (
	github.com/yubo/apiserver v0.0.0-00010101000000-000000000000
	github.com/yubo/golib v0.0.3-0.20220619092530-4b2095953f3f
	golang.org/x/net v0.0.0-20220225172249-27dd8689420f
	k8s.io/klog/v2 v2.60.1
)
