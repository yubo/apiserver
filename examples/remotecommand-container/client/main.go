package main

import (
	"fmt"
	"os"

	"github.com/yubo/apiserver/pkg/client"
	"github.com/yubo/apiserver/pkg/proc"
	"github.com/yubo/client-go/rest"
)

func main() {
	os.Exit(proc.PrintErrln(run()))
}

func run() error {
	config := &rest.Config{Host: "127.0.0.1:8080"}

	containerId, name, args, err := getArgs()
	if err != nil {
		return err
	}

	return client.NewExecClient(config, "POST", "/remotecommand/exec").
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
