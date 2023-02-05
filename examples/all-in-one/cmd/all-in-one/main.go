package main

import (
	"os"

	"github.com/yubo/apiserver/components/cli"
)

func main() {
	command := newServerCmd()
	code := cli.Run(command)
	os.Exit(code)
}
