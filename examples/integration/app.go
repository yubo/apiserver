package main

import (
	"context"

	"github.com/spf13/cobra"
	"github.com/yubo/apiserver/examples/integration/authn"
	"github.com/yubo/apiserver/examples/integration/authz"
	"github.com/yubo/apiserver/examples/integration/session"
	"github.com/yubo/apiserver/examples/integration/tracing"
	"github.com/yubo/apiserver/examples/integration/user"
	"github.com/yubo/apiserver/pkg/options"
	"github.com/yubo/golib/configer"
	"github.com/yubo/golib/proc"
	"k8s.io/klog/v2"

	// authz's submodule, should be loaded before the authz module
	_ "github.com/yubo/apiserver/pkg/authorization/abac/register"
	_ "github.com/yubo/apiserver/pkg/authorization/alwaysallow/register"
	_ "github.com/yubo/apiserver/pkg/authorization/alwaysdeny/register"
	_ "github.com/yubo/apiserver/pkg/authorization/register"

	// TODO
	//_ "github.com/yubo/apiserver/pkg/authorization/rbac/register"
	// TODO
	//_ "github.com/yubo/apiserver/pkg/authorization/webhook/register"

	// authn
	_ "github.com/yubo/apiserver/pkg/authentication/register"
	_ "github.com/yubo/apiserver/pkg/authentication/session/register"
	_ "github.com/yubo/apiserver/pkg/authentication/token/bootstrap/register"
	_ "github.com/yubo/apiserver/pkg/authentication/token/oidc/register"
	_ "github.com/yubo/apiserver/pkg/authentication/token/tokenfile/register"

	// TODO
	//_ "github.com/yubo/apiserver/pkg/authentication/serviceaccount/register"
	// TODO
	//_ "github.com/yubo/apiserver/pkg/authentication/webhook/register"

	_ "github.com/yubo/apiserver/pkg/apiserver/register"
	_ "github.com/yubo/apiserver/pkg/db/register"
	_ "github.com/yubo/apiserver/pkg/debug/register"
	_ "github.com/yubo/apiserver/pkg/grpcserver/register"
	_ "github.com/yubo/apiserver/pkg/rest/swagger/register"
	_ "github.com/yubo/apiserver/pkg/session/register"
	_ "github.com/yubo/apiserver/pkg/tracing/register"
	_ "github.com/yubo/golib/orm/sqlite"
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
	ctx = proc.WithConfigOps(ctx, configer.WithFlagOptions(true, false, 5))

	cmd := proc.NewRootCmd(ctx)
	cmd.AddCommand(options.NewVersionCmd())

	return cmd
}

func start(ctx context.Context) error {
	klog.Info("start")

	if err := session.New(ctx).Start(); err != nil {
		return err
	}
	if err := tracing.New(ctx).Start(); err != nil {
		return err
	}
	if err := user.New(ctx).Start(); err != nil {
		return err
	}
	if err := authn.New(ctx).Start(); err != nil {
		return err
	}
	if err := authz.New(ctx).Start(); err != nil {
		return err
	}

	return nil
}

func stop(ctx context.Context) error {
	klog.Info("stop")
	return nil
}
