module examples/websocket-watch

go 1.16

replace github.com/yubo/apiserver => ../..

require (
	github.com/yubo/apiserver v0.0.0-00010101000000-000000000000
	github.com/yubo/golib v0.0.3-0.20220825061925-f4cd420e40b5
	golang.org/x/net v0.0.0-20220225172249-27dd8689420f
	k8s.io/klog/v2 v2.70.1
)
