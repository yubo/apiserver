module examples/authn-x509

go 1.16

replace github.com/yubo/apiserver => ../..

require (
	github.com/yubo/apiserver v0.0.0-00010101000000-000000000000
	github.com/yubo/golib v0.0.3-0.20220629111701-7d4b450bd267
	golang.org/x/net v0.0.0-20220225172249-27dd8689420f // indirect
	k8s.io/klog/v2 v2.60.1
)