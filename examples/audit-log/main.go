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
		Path:   "/api",
		Server: srv,
		Routes: []server.WsRoute{
			{Method: "POST", SubPath: "/users", Handle: hw}, // RequestResponse
			{Method: "GET", SubPath: "/tokens", Handle: hw}, // metadata
		},
	})
	server.WsRouteBuild(&server.WsOption{
		Path:   "/static",
		Server: srv,
		Routes: []server.WsRoute{
			{Method: "GET", SubPath: "/hw", Handle: hw}, // none
		},
	})

	return nil
}

func hw(w http.ResponseWriter, req *http.Request) ([]byte, error) {
	return []byte(req.URL.Path + ": hello, world\n"), nil
}
