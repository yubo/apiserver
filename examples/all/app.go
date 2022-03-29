package main

import (
	"context"

	"github.com/spf13/cobra"
	"github.com/yubo/apiserver/examples/all/authn"
	"github.com/yubo/apiserver/examples/all/authz"
	"github.com/yubo/apiserver/examples/all/session"
	"github.com/yubo/apiserver/examples/all/trace"
	"github.com/yubo/apiserver/examples/all/user"
	"github.com/yubo/apiserver/pkg/options"
	"github.com/yubo/golib/proc"
	"k8s.io/klog/v2"

	_ "github.com/yubo/golib/orm/mysql"
	_ "github.com/yubo/golib/orm/sqlite"

	// authz's submodule, should be loaded before the authz module
	_ "github.com/yubo/apiserver/pkg/authorization/register"
	_ "github.com/yubo/apiserver/plugin/authorizer/abac/register"
	_ "github.com/yubo/apiserver/plugin/authorizer/alwaysallow/register"
	_ "github.com/yubo/apiserver/plugin/authorizer/alwaysdeny/register"
	_ "github.com/yubo/apiserver/plugin/authorizer/rbac/register"

	// TODO
	//_ "github.com/yubo/apiserver/plugin/authorizer/webhook/register"

	// authn
	_ "github.com/yubo/apiserver/pkg/authentication/register"
	// 1. headerrequest
	_ "github.com/yubo/apiserver/plugin/authenticator/headerrequest/register"
	// 2. x509
	_ "github.com/yubo/apiserver/plugin/authenticator/x509/register"
	// 3.session
	_ "github.com/yubo/apiserver/plugin/authenticator/session/register"
	// 4. tokenfile
	_ "github.com/yubo/apiserver/plugin/authenticator/token/tokenfile/register"
	// 5. service account file <TODO>
	// 6. service account issuer <TODO>
	// 7. bootstrap
	_ "github.com/yubo/apiserver/plugin/authenticator/token/bootstrap/register"
	// 8. OIDC
	_ "github.com/yubo/apiserver/plugin/authenticator/token/oidc/register"
	// 9. webhook
	_ "github.com/yubo/apiserver/plugin/authenticator/token/webhook/register"

	_ "github.com/yubo/apiserver/pkg/audit/register"
	_ "github.com/yubo/apiserver/pkg/db/register"
	_ "github.com/yubo/apiserver/pkg/grpcserver/register"
	_ "github.com/yubo/apiserver/pkg/models/register"
	_ "github.com/yubo/apiserver/pkg/server/register"
	_ "github.com/yubo/apiserver/pkg/session/register"
	_ "github.com/yubo/apiserver/pkg/traces/register"
)

const (
	AppName    = "all.example.apiserver"
	moduleName = "all.example.apiserver"
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
	options.InstallReporter()

	cmd := proc.NewRootCmd(proc.WithHooks(hookOps...))
	cmd.AddCommand(options.NewVersionCmd())

	return cmd
}

func start(ctx context.Context) error {
	klog.Info("start")

	if err := session.New(ctx).Start(); err != nil {
		return err
	}
	if err := trace.New(ctx).Start(); err != nil {
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
