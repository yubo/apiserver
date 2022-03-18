module github.com/yubo/apiserver/examples/remotecommand-native

go 1.16

replace github.com/yubo/apiserver => ../..

require (
	github.com/emicklei/go-restful/v3 v3.7.4
	github.com/yubo/apiserver v0.0.0-00010101000000-000000000000
	github.com/yubo/golib v0.0.3-0.20220318153846-9a4409366d5f
	k8s.io/klog/v2 v2.60.0
)
