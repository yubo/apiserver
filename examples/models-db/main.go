package main

import (
	"context"
	"os"

	"examples/models/api"
	"examples/models/models"

	server "github.com/yubo/apiserver/pkg/server/module"
	"github.com/yubo/apiserver/components/cli"
	"github.com/yubo/apiserver/pkg/proc"
	"k8s.io/klog/v2"

	// models
	_ "github.com/yubo/apiserver/pkg/models/register"
	// db
	_ "github.com/yubo/apiserver/pkg/db/register"
	_ "github.com/yubo/golib/orm/mysql"
	_ "github.com/yubo/golib/orm/sqlite"
)

const (
	moduleName = "db.models.examples"
)

var (
	hookOps = []v1.HookOps{{
		Hook:     start,
		Owner:    moduleName,
		HookNum:  v1.ACTION_START,
		Priority: v1.PRI_MODULE,
	}}
)

func main() {
	cmd := proc.NewRootCmd(server.WithoutTLS(), proc.WithHooks(hookOps...))
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

	if err := m.Create(ctx, secret); err != nil {
		return err
	}

	klog.InfoS("create", "name", secret.Name, "data", secret.Data)

	if e, err := m.Get(ctx, "name=token-test"); err != nil {
		return err
	} else {
		klog.InfoS("get", "name", e.Name, "data", e.Data)
	}

	secret.Data = "2"
	if err := m.Update(ctx, secret); err != nil {
		return err
	}

	if e, err := m.Get(ctx, "name=token-test"); err != nil {
		return err
	} else {
		klog.InfoS("get", "name", e.Name, "data", e.Data)
	}

	return nil
}
