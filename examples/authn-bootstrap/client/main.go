package main

import (
	"context"
	"flag"
	"os"

	"github.com/yubo/apiserver/pkg/authentication/user"
	"github.com/yubo/apiserver/pkg/util/webhook"
	"github.com/yubo/golib/api"
	"github.com/yubo/golib/scheme"
	"github.com/yubo/golib/util/wait"
	"k8s.io/klog/v2"
)

var (
	retryBackoff = wait.Backoff{
		Duration: api.NewDuration("5ms"),
		Factor:   1.5,
		Jitter:   0.2,
		Steps:    5,
	}
)

func main() {
	configFile := flag.String("conf", "./client.conf", "webhook config path")
	flag.Parse()

	if err := func() error {
		w, err := webhook.NewGenericWebhook(scheme.Codec, *configFile, retryBackoff, nil)
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
