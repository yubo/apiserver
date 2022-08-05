package main

import (
	"fmt"
	"os"

	rest "github.com/yubo/apiserver/pkg/client"
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
	config := &rest.Config{Host: "127.0.0.1:8080"}

	containerId, name, args, err := getArgs()
	if err != nil {
		return err
	}

	return cmdcli.NewExecClient(config, "POST", "/remotecommand/exec").
		Command(name, args...).
		Container(containerId).
		Run()
}

func getArgs() (containerId, name string, arg []string, err error) {
	if len(os.Args) < 3 {
		err = fmt.Errorf("Usage: %s <container id> <name> <args> ...", os.Args[0])
		return
	}

	containerId = os.Args[1]
	name = os.Args[2]

	if len(os.Args) > 3 {
		arg = os.Args[3:]
	}

	return
}
