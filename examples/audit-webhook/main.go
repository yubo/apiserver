package main

import (
	"context"
	"net/http"
	"os"

	"github.com/yubo/apiserver/components/cli"
	"github.com/yubo/apiserver/components/dbus"
	"github.com/yubo/apiserver/pkg/proc"
	"github.com/yubo/apiserver/pkg/rest"

	// http
	server "github.com/yubo/apiserver/pkg/server/module"
	_ "github.com/yubo/apiserver/pkg/server/register"

	// audit
	_ "github.com/yubo/apiserver/pkg/audit/register"
)

func main() {
	command := proc.NewRootCmd(server.WithoutTLS(), proc.WithRun(start))
	code := cli.Run(command)
	os.Exit(code)
}

func start(ctx context.Context) error {
	srv, err := dbus.GetAPIServer()
	if err != nil {
		return err
	}

	rest.WsRouteBuild(&rest.WsOption{
		Path:               "/audit",
		GoRestfulContainer: srv,
		Routes: []rest.WsRoute{
			{Method: "POST", SubPath: "/hello", Handle: hw},
		},
	})

	return nil
}

func hw(w http.ResponseWriter, req *http.Request) (string, error) {
	return "hello, world", nil
}
