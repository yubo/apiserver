package main

import (
	"context"
	"os"

	"github.com/spf13/cobra"
	"github.com/yubo/apiserver/example/session"
	"github.com/yubo/apiserver/example/tracing"
	"github.com/yubo/apiserver/example/user"
	"github.com/yubo/apiserver/pkg/options"
	"github.com/yubo/golib/proc"
	"k8s.io/klog/v2"

	_ "github.com/yubo/apiserver/pkg/apiserver/register"
	_ "github.com/yubo/apiserver/pkg/authentication/register"
	_ "github.com/yubo/apiserver/pkg/authorization/register"
	_ "github.com/yubo/apiserver/pkg/db/register"
	_ "github.com/yubo/apiserver/pkg/debug/register"
	_ "github.com/yubo/apiserver/pkg/grpcserver/register"
	_ "github.com/yubo/apiserver/pkg/session/register"
	_ "github.com/yubo/apiserver/pkg/swagger/register"
	_ "github.com/yubo/apiserver/pkg/tracing/register"
	_ "github.com/yubo/golib/orm/sqlite"

	// authn
	_ "github.com/yubo/apiserver/pkg/authentication/register/bootstrap"
	_ "github.com/yubo/apiserver/pkg/authentication/register/oidc"
	_ "github.com/yubo/apiserver/pkg/authentication/register/serviceaccount"
	_ "github.com/yubo/apiserver/pkg/authentication/register/tokenfile"
	_ "github.com/yubo/apiserver/pkg/authentication/register/webhook"
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

	if err := session.New(ctx).Start(); err != nil {
		return err
	}
	if err := tracing.New(ctx).Start(); err != nil {
		return err
	}
	if err := user.New(ctx).Start(); err != nil {
		return err
	}

	return nil
}

func stop(ops *proc.HookOps) error {
	klog.Info("stop")
	return nil
}
