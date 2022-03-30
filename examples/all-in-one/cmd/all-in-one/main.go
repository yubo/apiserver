package main

import (
	"os"

	"github.com/yubo/golib/cli"
)

func main() {
	command := newServerCmd()
	code := cli.Run(command)
	os.Exit(code)
}
