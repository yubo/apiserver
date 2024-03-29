package main

import (
	"fmt"
	"os"

	"github.com/yubo/apiserver/pkg/client"
	"github.com/yubo/apiserver/pkg/proc"
	"github.com/yubo/client-go/rest"
	"github.com/yubo/golib/scheme"
)

func main() {
	os.Exit(proc.PrintErrln(run()))
}

func run() error {
	name, args, err := getArgs()
	if err != nil {
		return err
	}

	config := &rest.Config{
		Host:          "127.0.0.1:8080",
		ContentConfig: rest.ContentConfig{NegotiatedSerializer: scheme.NegotiatedSerializer},
	}

	return client.NewExecClient(config, "POST", "/remotecommand/exec").
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
