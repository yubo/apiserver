package main

import (
	"context"
	"examples/rest/user"
	"os"

	"github.com/yubo/apiserver/components/cli"
	"github.com/yubo/apiserver/pkg/proc"
	v1 "github.com/yubo/apiserver/pkg/proc/api/v1"
	server "github.com/yubo/apiserver/pkg/server/module"

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
	hookOps = []v1.HookOps{{
		Hook:     start,
		Owner:    moduleName,
		HookNum:  v1.ACTION_START,
		Priority: v1.PRI_MODULE,
	}}
)

func main() {
	command := proc.NewRootCmd(server.WithoutTLS(), proc.WithHooks(hookOps...))
	code := cli.Run(command)
	os.Exit(code)
}

func start(ctx context.Context) error {

	user.New(ctx).Install()

	return nil
}
