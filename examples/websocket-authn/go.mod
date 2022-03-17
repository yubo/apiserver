module github.com/yubo/apiserver/examples/websocket-authn

go 1.16

replace github.com/yubo/apiserver => ../..

require (
	github.com/yubo/apiserver v0.0.0-00010101000000-000000000000
	github.com/yubo/golib v0.0.2-0.20220109150524-29ec97838a83
	golang.org/x/net v0.0.0-20220225172249-27dd8689420f
	k8s.io/klog/v2 v2.60.0
)
