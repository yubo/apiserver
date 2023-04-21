module examples/custom-configer-cmds

go 1.20

replace github.com/yubo/golib => ../../../golib

replace github.com/yubo/apiserver => ../../../apiserver

require (
	github.com/spf13/cobra v1.4.0
	github.com/yubo/apiserver v0.0.0-00010101000000-000000000000
	github.com/yubo/golib v0.0.3-0.20230413153058-1831b5929edc
	k8s.io/klog/v2 v2.80.1
)
