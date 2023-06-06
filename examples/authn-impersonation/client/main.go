package main

import (
	"context"
	"flag"
	"os"
	"time"

	"github.com/yubo/apiserver/pkg/authentication/user"
	"github.com/yubo/apiserver/pkg/util/webhook"
	"k8s.io/klog/v2"
)

func main() {
	configFile := flag.String("conf", "./client.conf", "webhook config path")
	flag.Parse()

	if err := func() error {
		w, err := webhook.NewWebhook(*configFile, 5*time.Millisecond)
		if err != nil {
			return err
		}

		resp := user.DefaultInfo{}
		err = w.RestClient.Get().Do(context.Background()).Into(&resp)
		if err != nil {
			return err
		}

		klog.InfoS("webhook", "resp", resp)

		return nil
	}(); err != nil {
		klog.Error(err)
		os.Exit(1)
	}
}
