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
	cmd := proc.NewRootCmd(proc.WithRun(start))
	code := cli.Run(cmd)
	os.Exit(code)
}

func start(ctx context.Context) error {
	srv, err := dbus.GetAPIServer()
	if err != nil {
		return err
	}
	server.WsRouteBuild(&server.WsOption{
		Path:   "/api/v1",
		Server: srv,
		Routes: []server.WsRoute{{
			Method:  "GET",
			SubPath: "/users",
			Consume: server.MIME_ALL,
			Handle:  handle,
		}},
	})

	return nil
}

func handle(w http.ResponseWriter, req *http.Request) ([]byte, error) {
	return []byte("OK\n"), nil
}
