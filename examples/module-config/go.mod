module examples/module-config

go 1.16

replace github.com/yubo/apiserver => ../..

require (
	github.com/spf13/cobra v1.4.0
	github.com/yubo/golib v0.0.3-0.20220619092530-4b2095953f3f
	sigs.k8s.io/yaml v1.3.0
)
