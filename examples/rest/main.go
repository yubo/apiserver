package main

import (
	"context"
	"examples/rest/user"
	"os"

	"github.com/yubo/apiserver/components/cli"
	"github.com/yubo/apiserver/pkg/proc"

	_ "github.com/yubo/apiserver/pkg/db/register"
	_ "github.com/yubo/apiserver/pkg/server/register"
	_ "github.com/yubo/golib/orm/mysql"
	_ "github.com/yubo/golib/orm/sqlite"
)

func main() {
	cmd := proc.NewRootCmd(proc.WithRun(start))
	code := cli.Run(cmd)
	os.Exit(code)
}

func start(ctx context.Context) error {
	user.New(ctx).Install()

	return nil
}
