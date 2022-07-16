package main

import (
	"context"
	"fmt"
	"os"

	"examples/rest/routes"

	"github.com/yubo/apiserver/pkg/options"
	server "github.com/yubo/apiserver/pkg/server/module"
	"github.com/yubo/golib/cli"
	"github.com/yubo/golib/proc"

	_ "github.com/yubo/apiserver/pkg/models/register"
	_ "github.com/yubo/apiserver/pkg/server/register"

	_ "github.com/yubo/apiserver/pkg/db/register"
	_ "github.com/yubo/golib/orm/mysql"
	_ "github.com/yubo/golib/orm/sqlite"
)

const (
	moduleName = "rest.examples"
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
	command := proc.NewRootCmd(server.WithoutTLS(), proc.WithHooks(hookOps...))
	code := cli.Run(command)
	os.Exit(code)
}

func start(ctx context.Context) error {
	http, ok := options.APIServerFrom(ctx)
	if !ok {
		return fmt.Errorf("unable to get http server from the context")
	}

	routes.InstallUser(http)

	return nil
}
