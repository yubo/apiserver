module examples/websocket-watch

go 1.16

replace github.com/yubo/apiserver => ../..

replace github.com/yubo/golib => ../../../golib

replace github.com/yubo/client-go => ../../../client-go

require (
	github.com/yubo/apiserver v0.0.0-00010101000000-000000000000
	github.com/yubo/golib v0.0.3-0.20220902030005-7f15ca001a44
	golang.org/x/net v0.5.0
	k8s.io/klog/v2 v2.80.1
)
