package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/yubo/apiserver/pkg/options"
	"github.com/yubo/golib/net/session"
	"github.com/yubo/golib/proc"
	"k8s.io/klog/v2"

	_ "github.com/yubo/apiserver/example/session"
	_ "github.com/yubo/apiserver/example/tracing"
	_ "github.com/yubo/apiserver/modules/apiserver"
	_ "github.com/yubo/apiserver/modules/authorization"
	_ "github.com/yubo/apiserver/modules/db"
	_ "github.com/yubo/apiserver/modules/debug"
	_ "github.com/yubo/apiserver/modules/grpcserver"
	_ "github.com/yubo/apiserver/modules/swagger"
	_ "github.com/yubo/apiserver/modules/tracing"
	_ "github.com/yubo/apiserver/pkg/authentication/module"
	_ "github.com/yubo/apiserver/pkg/session/module"
	_ "github.com/yubo/golib/orm/sqlite"

	"github.com/yubo/apiserver/example/user"
)

const (
	AppName    = "helo"
	moduleName = "helo.main"
)

var (
	hookOps = []proc.HookOps{{
		Hook:     start,
		Owner:    moduleName,
		HookNum:  proc.ACTION_START,
		Priority: proc.PRI_MODULE,
	}, {
		Hook:     stop,
		Owner:    moduleName,
		HookNum:  proc.ACTION_STOP,
		Priority: proc.PRI_MODULE,
	}}
)

func newServerCmd() *cobra.Command {
	proc.RegisterHooks(hookOps)
	options.InstallReporter()

	ctx := context.Background()
	ctx = proc.WithName(ctx, AppName)
	ctx = proc.WithConfigOps(ctx) //config.WithBaseBytes2("http", app.DefaultOptions),

	cmd := proc.NewRootCmd(ctx, os.Args[1:])
	cmd.AddCommand(options.NewVersionCmd())

	return cmd
}

func start(ops *proc.HookOps) error {
	klog.Info("start")

	ctx := ops.Context()

	db, ok := options.DBFrom(ctx)
	if !ok {
		return fmt.Errorf("unable to get db")
	}

	if err := db.ExecRows([]byte(session.CREATE_TABLE_SQLITE)); err != nil {
		return err
	}
	if err := db.ExecRows([]byte(user.CREATE_TABLE_SQLITE)); err != nil {
		return err
	}

	return nil
}

func stop(ops *proc.HookOps) error {
	klog.Info("stop")
	return nil
}
