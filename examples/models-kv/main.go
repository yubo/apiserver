package main

import (
	"context"
	"os"

	"examples/models-kv/api"
	"examples/models-kv/models"

	"github.com/yubo/apiserver/components/cli"
	"github.com/yubo/apiserver/pkg/proc"
	"k8s.io/klog/v2"

	// db
	_ "github.com/yubo/apiserver/pkg/db/register"
	_ "github.com/yubo/golib/orm/mysql"
	_ "github.com/yubo/golib/orm/sqlite"
)

func main() {
	cmd := proc.NewRootCmd(proc.WithRun(start))
	code := cli.Run(cmd)
	os.Exit(code)
}

func start(ctx context.Context) error {
	defer proc.Shutdown()

	m := models.NewDemo()

	secret := &api.Demo{
		Name: "token-test",
		Data: "1",
	}

	if e, err := m.Create(ctx, secret); err != nil {
		return err
	} else {
		klog.InfoS("create", "name", e.Name, "data", e.Data)
	}

	if e, err := m.Get(ctx, "token-test"); err != nil {
		return err
	} else {
		klog.InfoS("get", "name", e.Name, "data", e.Data)
	}

	secret.Data = "2"
	if e, err := m.Update(ctx, secret); err != nil {
		return err
	} else {
		klog.InfoS("get", "name", e.Name, "data", e.Data)
	}

	return nil
}
