package main

import (
	"fmt"
	"os"

	"github.com/yubo/apiserver/pkg/cmdcli"
	"github.com/yubo/apiserver/pkg/rest"
	"github.com/yubo/golib/scheme"
	"k8s.io/klog/v2"
)

func main() {
	if err := run(); err != nil {
		klog.Error(err)
		os.Exit(1)
	}
}

func run() error {
	name, args, err := getArgs()
	if err != nil {
		return err
	}

	config := &rest.Config{
		Host:          "127.0.0.1:8080",
		ContentConfig: rest.ContentConfig{NegotiatedSerializer: scheme.Codecs},
	}

	return cmdcli.NewExecClient(config, "POST", "/remotecommand/exec").
		Command(name, args...).
		Run()
}

func getArgs() (name string, args []string, err error) {
	if len(os.Args) < 2 {
		err = fmt.Errorf("Usage: %s <name> <args> ...", os.Args[0])
		return
	}

	name = os.Args[1]

	if len(os.Args) > 2 {
		args = os.Args[2:]
	}

	return
}
