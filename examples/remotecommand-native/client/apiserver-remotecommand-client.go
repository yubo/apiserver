package main

import (
	"fmt"
	"os"

	"github.com/yubo/apiserver/pkg/cmdcli"
	"k8s.io/klog/v2"
)

func main() {
	if err := run(); err != nil {
		klog.Error(err)
		os.Exit(1)
	}
}

func run() error {
	name, arg, err := getArgs()
	if err != nil {
		return err
	}

	return cmdcli.NewExecClient(
		"POST",
		"127.0.0.1:8080",
		"/exec").
		Exec(name, arg...)
}

func getArgs() (name string, arg []string, err error) {
	if len(os.Args) < 2 {
		err = fmt.Errorf("Usage: %s <name> <args> ...", os.Args[0])
		return
	}

	name = os.Args[1]

	if len(os.Args) > 2 {
		arg = os.Args[2:]
	}

	return
}
