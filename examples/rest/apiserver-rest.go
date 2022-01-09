package main

import (
	"context"
	"fmt"
	"os"

	"github.com/yubo/apiserver/examples/rest/routes"
	"github.com/yubo/apiserver/pkg/options"
	server "github.com/yubo/apiserver/pkg/server/module"
	"github.com/yubo/golib/logs"
	"github.com/yubo/golib/orm"
	"github.com/yubo/golib/proc"

	_ "github.com/yubo/apiserver/pkg/models/register"
	_ "github.com/yubo/apiserver/pkg/server/register"

	_ "github.com/yubo/golib/orm/mysql"
	_ "github.com/yubo/golib/orm/sqlite"
)

const (
	moduleName = "example.rest.apiserver"
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
	logs.InitLogs()
	defer logs.FlushLogs()
	orm.DEBUG = true

	if err := server.NewRootCmdWithoutTLS().Execute(); err != nil {
		os.Exit(1)
	}
}

func start(ctx context.Context) error {
	http, ok := options.APIServerFrom(ctx)
	if !ok {
		return fmt.Errorf("unable to get http server from the context")
	}

	routes.InstallUser(http)

	return nil
}

func init() {
	proc.RegisterHooks(hookOps)
}
