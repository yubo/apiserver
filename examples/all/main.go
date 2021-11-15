package main

import (
	"os"

	"github.com/yubo/golib/logs"
)

func main() {
	logs.InitLogs()
	defer logs.FlushLogs()

	if err := newServerCmd().Execute(); err != nil {
		os.Exit(1)
	}
}
