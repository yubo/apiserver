package main

import (
	"fmt"
	"os"

	restclient "github.com/yubo/apiserver/pkg/client"
	"github.com/yubo/apiserver/pkg/cmdcli"
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
	if len(os.Args) < 1 {
		return fmt.Errorf("Usage: %s <session-id>", os.Args[0])
	}

	config := &restclient.Config{
		Host:          "127.0.0.1:8080",
		ContentConfig: restclient.ContentConfig{NegotiatedSerializer: scheme.Codecs},
	}

	return cmdcli.NewExecClient(config, "POST", "/remotecommand/attach").
		Attach(os.Args[1]).
		Run()
}
