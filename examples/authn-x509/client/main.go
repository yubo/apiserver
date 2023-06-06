package main

import (
	"context"
	"flag"
	"os"
	"time"

	"github.com/yubo/apiserver/pkg/authentication/user"
	"github.com/yubo/apiserver/pkg/proc"
	"github.com/yubo/apiserver/pkg/util/webhook"
	"k8s.io/klog/v2"
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
	w, err := webhook.NewWebhook(config, 5*time.Millisecond)
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
