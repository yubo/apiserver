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
	if len(os.Args) < 1 {
		return fmt.Errorf("Usage: %s <session-id>", os.Args[0])
	}

	config := &rest.Config{
		Host:          "127.0.0.1:8080",
		ContentConfig: rest.ContentConfig{NegotiatedSerializer: scheme.Codecs},
	}

	return client.NewExecClient(config, "POST", "/remotecommand/attach").
		Attach(os.Args[1]).
		Run()
}
