module github.com/yubo/apiserver/examples/models

go 1.16

replace github.com/yubo/apiserver => ../..

replace github.com/yubo/golib => ../../../golib

require (
	github.com/go-openapi/spec v0.20.5 // indirect
	github.com/yubo/apiserver v0.0.0-00010101000000-000000000000
	github.com/yubo/golib v0.0.3-0.20220328093426-c3a2f6ed6613
	k8s.io/klog/v2 v2.60.1
)
