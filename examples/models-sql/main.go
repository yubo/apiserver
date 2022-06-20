package main

import (
	"context"
	"os"

	"examples/models/api"
	"examples/models/models"

	server "github.com/yubo/apiserver/pkg/server/module"
	"github.com/yubo/golib/cli"
	"github.com/yubo/golib/proc"
	"k8s.io/klog/v2"

	// models
	_ "github.com/yubo/apiserver/pkg/models/register"
	// db
	_ "github.com/yubo/apiserver/pkg/db/register"
	_ "github.com/yubo/golib/orm/mysql"
	_ "github.com/yubo/golib/orm/sqlite"
)

const (
	moduleName = "models.apiserver"
)

var (
	hookOps = []proc.HookOps{{
		Hook:     start,
		Owner:    moduleName,
		HookNum:  proc.ACTION_START,
		Priority: proc.PRI_MODULE,
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

	if e, err := m.Get(ctx, "token-test"); err != nil {
		return err
	} else {
		klog.InfoS("get", "name", e.Name, "data", e.Data)
	}

	secret.Data = "2"
	if err := m.Update(ctx, secret); err != nil {
		return err
	}

	if e, err := m.Get(ctx, "token-test"); err != nil {
		return err
	} else {
		klog.InfoS("get", "name", e.Name, "data", e.Data)
	}

	return nil
}
