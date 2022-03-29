package main

import (
	"context"
	"os"

	"github.com/spf13/cobra"
	"github.com/yubo/apiserver/examples/all-in-one/authn"
	"github.com/yubo/apiserver/examples/all-in-one/authz"
	"github.com/yubo/apiserver/examples/all-in-one/session"
	"github.com/yubo/apiserver/examples/all-in-one/trace"
	"github.com/yubo/apiserver/examples/all-in-one/user"
	"github.com/yubo/apiserver/pkg/version/reporter"
	"github.com/yubo/golib/cli"
	"github.com/yubo/golib/proc"
	"k8s.io/klog/v2"

	_ "github.com/yubo/apiserver/examples/all-in-one/components"
)

const (
	moduleName = "all.example.apiserver"
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
	command := newServerCmd()
	code := cli.Run(command)
	os.Exit(code)
}

func newServerCmd() *cobra.Command {
	cmd := proc.NewRootCmd(proc.WithHooks(hookOps...))
	cmd.AddCommand(reporter.NewVersionCmd())

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
