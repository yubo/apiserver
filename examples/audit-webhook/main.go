package main

import (
	"context"
	"net/http"
	"os"

	"github.com/yubo/apiserver/components/cli"
	"github.com/yubo/apiserver/components/dbus"
	"github.com/yubo/apiserver/pkg/proc"
	"github.com/yubo/apiserver/pkg/server"

	// http
	_ "github.com/yubo/apiserver/pkg/server/register"
)

func main() {
	command := proc.NewRootCmd(proc.WithRun(start))
	code := cli.Run(command)
	os.Exit(code)
}

func start(ctx context.Context) error {
	srv, err := dbus.GetAPIServer()
	if err != nil {
		return err
	}

	server.WsRouteBuild(&server.WsOption{
		Path:   "/audit",
		Server: srv,
		Routes: []server.WsRoute{
			{Method: "POST", SubPath: "/hello", Handle: hw},
		},
	})

	return nil
}

func hw(w http.ResponseWriter, req *http.Request) (string, error) {
	return "hello, world", nil
}
