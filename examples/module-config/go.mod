module examples/module-config

go 1.16

replace github.com/yubo/apiserver => ../..

require (
	github.com/spf13/cobra v1.4.0 // indirect
	github.com/yubo/golib v0.0.3-0.20220629111701-7d4b450bd267
	sigs.k8s.io/yaml v1.3.0
)
