package main

import (
	"github.com/yubo/golib/staging/logs"
	"k8s.io/klog/v2"
)

func main() {
	logs.InitLogs()
	defer logs.FlushLogs()

	if err := newServerCmd().Execute(); err != nil {
		klog.Error(err)
	}
}
