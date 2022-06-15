package main

import (
	"context"
	"os"

	"github.com/yubo/apiserver/pkg/util/webhook"
	"github.com/yubo/golib/api"
	"github.com/yubo/golib/scheme"
	"github.com/yubo/golib/util/wait"
	"k8s.io/klog/v2"
)

var (
	kubeConfigFile = "./kube.conf"
	retryBackoff   = wait.Backoff{
		Duration: api.NewDuration("5ms"),
		Factor:   1.5,
		Jitter:   0.2,
		Steps:    5,
	}
)

type Input struct {
	X int
}

type Output struct {
	X int
}

func main() {
	if err := func() error {
		w, err := webhook.NewGenericWebhook(scheme.Codecs, kubeConfigFile, retryBackoff, nil)
		if err != nil {
			return err
		}

		output := Output{}
		input := Input{1}
		err = w.RestClient.Post().Body(&input).Do(context.Background()).Into(&output)
		if err != nil {
			return err
		}

		klog.InfoS("webhook", "input", input, "output", output)

		return nil
	}(); err != nil {
		klog.Error(err)
		os.Exit(1)
	}
}
