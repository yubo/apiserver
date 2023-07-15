package main

import (
	"context"
	"errors"
	"os"

	"github.com/yubo/apiserver/components/cli"
	"github.com/yubo/apiserver/pkg/proc"
	"k8s.io/klog/v2"

	_ "github.com/yubo/apiserver/components/logs/json/register"
)

func main() {
	code := cli.Run(proc.NewRootCmd(
		proc.WithRun(runLogger),
		proc.WithName("logger"),
		proc.WithoutLoop(),
	))
	os.Exit(code)
}

func runLogger(ctx context.Context) error {
	klog.Infof("Log using Infof, key: %s", "value")
	klog.InfoS("Log using InfoS", "key", "value")
	err := errors.New("fail")
	klog.Errorf("Log using Errorf, err: %v", err)
	klog.ErrorS(err, "Log using ErrorS")
	data := SensitiveData{Key: "secret"}
	klog.Infof("Log with sensitive key, data: %q", data)
	for i := 0; i < 10; i++ {
		klog.V(klog.Level(i)).Infof("level %d info", i)
	}

	return nil
}

type SensitiveData struct {
	Key string `json:"key" datapolicy:"secret-key"`
}
