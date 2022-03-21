package main

import (
	"context"
	"fmt"
	"os"

	"github.com/yubo/apiserver/pkg/options"
	http "github.com/yubo/apiserver/pkg/server/module"
	"github.com/yubo/golib/proc"

	// http server
	_ "github.com/yubo/apiserver/pkg/server/register"

	// grpc server
	_ "github.com/yubo/apiserver/pkg/grpcserver/register"

	// tracing
	_ "github.com/yubo/apiserver/pkg/tracing/register"
)

const (
	moduleName = "example.tracing.apiserver"
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
	//orm.DEBUG = true

	if err := http.NewRootCmdWithoutTLS().Execute(); err != nil {
		os.Exit(1)
	}
}

func start(ctx context.Context) error {
	http, ok := options.APIServerFrom(ctx)
	if !ok {
		return fmt.Errorf("unable to get http server from the context")
	}

	demo{http}.install()

	return nil
}

func init() {
	proc.RegisterHooks(hookOps)
}
