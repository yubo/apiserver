package main

import (
	"context"
	"os"

	"github.com/yubo/apiserver/pkg/proc"
	"github.com/yubo/apiserver/pkg/server"
	"github.com/yubo/client-go/rest"
	"github.com/yubo/golib/scheme"
	"k8s.io/klog/v2"
)

func main() {
	os.Exit(proc.PrintErrln(run()))
}

// http.Flusher
func run() error {
	c, err := rest.RESTClientFor(
		&rest.Config{
			Host:          "127.0.0.1:8080",
			ContentConfig: rest.ContentConfig{NegotiatedSerializer: scheme.Codecs},
		},
	)
	if err != nil {
		return err
	}

	watching, err := c.Get().
		SetHeader("Content-Type", server.MIME_JSON).
		Prefix("hello").
		Watch(context.Background(), new(string))
	if err != nil {
		return err
	}

	ch := watching.ResultChan()
	for {
		got, ok := <-ch
		if !ok {
			klog.Info("unable to watching, exit")
			return nil
		}
		klog.InfoS("recv event", "type", got.Type, "object", *got.Object.(*string))
	}
}
