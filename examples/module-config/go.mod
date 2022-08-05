module examples/module-config

go 1.16

replace github.com/yubo/apiserver => ../..

require (
	github.com/spf13/cobra v1.4.0 // indirect
	github.com/yubo/golib v0.0.3-0.20220805044825-17febb0ab226
	sigs.k8s.io/yaml v1.3.0
)
