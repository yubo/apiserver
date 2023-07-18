package main

import (
	// set default config, must before the other modules
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
		Path:   "/hello",
		Server: srv,
		Routes: []server.WsRoute{
			{Method: "GET", SubPath: "/ro", Handle: handle},
			{Method: "GET", SubPath: "/deny", Handle: handle},
		},
	})

	return nil
}

func handle(w http.ResponseWriter, req *http.Request) ([]byte, error) {
	return []byte("hello\n"), nil
}
