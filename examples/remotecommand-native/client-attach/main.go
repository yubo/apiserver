package main

import (
	"fmt"
	"os"

	"github.com/yubo/apiserver/pkg/client"
	"github.com/yubo/client-go/rest"
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

	config := &rest.Config{
		Host:          "127.0.0.1:8080",
		ContentConfig: rest.ContentConfig{NegotiatedSerializer: scheme.Codecs},
	}

	return client.NewExecClient(config, "POST", "/remotecommand/attach").
		Attach(os.Args[1]).
		Run()
}
