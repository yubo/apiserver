package main

import (
	"context"
	"flag"
	"os"
	"time"

	"github.com/yubo/apiserver/pkg/authentication/user"
	"github.com/yubo/apiserver/pkg/proc"
	"github.com/yubo/apiserver/pkg/util/webhook"
	"github.com/yubo/client-go/transport"
	"k8s.io/klog/v2"
)

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

	cli := w.RestClient.Client
	cli.Transport = transport.NewAuthProxyRoundTripper(
		"tom",                    // user
		[]string{"dev", "admin"}, // groups
		//extras
		map[string][]string{
			"acme.com/project": {"some-project"},
			"scopes":           {"openid", "profile"},
		},
		cli.Transport,
	)

	var output user.DefaultInfo
	err = w.RestClient.Get().Do(context.Background()).Into(&output)
	if err != nil {
		return err
	}

	klog.InfoS("webhook", "output", output)

	return nil

}
