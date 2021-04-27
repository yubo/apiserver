package main

import (
	"os"

	"github.com/yubo/golib/staging/logs"
)

func main() {
	cmd := newServerCmd()

	logs.InitLogs()
	defer logs.FlushLogs()

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
