package main

import (
	"context"
	"os"
	"time"

	"github.com/yubo/apiserver/pkg/rest"
	"github.com/yubo/golib/scheme"
	"k8s.io/klog/v2"
)

// http.Flusher
func flusher() error {
	c, err := rest.RESTClientFor(
		&rest.Config{
			Host:          "127.0.0.1:8080",
			ContentConfig: rest.ContentConfig{NegotiatedSerializer: scheme.Codecs},
		},
	)
	if err != nil {
		return err
	}

	watching, err := c.Get().Prefix("hello").Watch(context.Background(), &time.Time{})
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
		klog.InfoS("recv event", "type", got.Type, "object", got.Object.(*time.Time))
	}
}

func main() {
	if err := flusher(); err != nil {
		klog.Error(err)
		os.Exit(1)
	}
}
