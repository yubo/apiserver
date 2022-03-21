package main

import (
	"context"
	"os"

	"github.com/yubo/apiserver/pkg/models"
	server "github.com/yubo/apiserver/pkg/server/module"
	"github.com/yubo/golib/api"
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

	m := models.NewSecret()

	secret := &api.Secret{
		ObjectMeta: api.ObjectMeta{
			Name: "bootstrap-token-test",
		},
		Type: "bootstrap.kubernetes.io/token",
	}

	if e, err := m.Create(ctx, secret); err != nil {
		return err
	} else {
		klog.InfoS("create", "name", e.Name, "type", e.Type)
	}

	if e, err := m.Get(ctx, "bootstrap-token-test"); err != nil {
		return err
	} else {
		klog.InfoS("get", "name", e.Name, "type", e.Type)
	}

	secret.Type = "bootstrap.kubernetes.io/token/2"
	if e, err := m.Update(ctx, secret); err != nil {
		return err
	} else {
		klog.InfoS("get", "name", e.Name, "type", e.Type)
	}

	return nil
}
