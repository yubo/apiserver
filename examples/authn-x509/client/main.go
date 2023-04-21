package main

import (
	"context"
	"flag"
	"os"

	"github.com/yubo/apiserver/pkg/authentication/user"
	"github.com/yubo/apiserver/pkg/proc"
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

type Input struct {
	X int
}

type Output struct {
	X    int
	User user.DefaultInfo
}

func main() {
	configFile := flag.String("conf", "./client.conf", "webhook config path")
	flag.Parse()

	os.Exit(proc.PrintErrln(run(*configFile)))
}

func run(config string) error {
	w, err := webhook.NewGenericWebhook(scheme.Codec, config, retryBackoff, nil)
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

}
