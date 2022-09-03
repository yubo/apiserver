module examples/authn-token-oidc

go 1.16

replace github.com/yubo/apiserver => ../..

require (
	github.com/yubo/apiserver v0.0.0-00010101000000-000000000000
	github.com/yubo/golib v0.0.3-0.20220902030005-7f15ca001a44
	gopkg.in/square/go-jose.v2 v2.5.1
	k8s.io/klog/v2 v2.70.1
)
