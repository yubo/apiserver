package main

import (
	"context"
	"net/http"
	"os"

	"github.com/yubo/apiserver/components/cli"
	"github.com/yubo/apiserver/components/dbus"
	"github.com/yubo/apiserver/pkg/proc"

	genericserver "github.com/yubo/apiserver/pkg/server"
	server "github.com/yubo/apiserver/pkg/server/module"
	_ "github.com/yubo/apiserver/pkg/server/register"
)

func main() {
	cmd := proc.NewRootCmd(server.WithoutTLS(), proc.WithRun(start))
	code := cli.Run(cmd)
	os.Exit(code)
}

func start(ctx context.Context) error {
	srv, err := dbus.GetAPIServer()
	if err != nil {
		return err
	}
	genericserver.WsRouteBuild(&genericserver.WsOption{
		Path:   "/hello",
		Server: srv,
		Routes: []genericserver.WsRoute{
			{Method: "GET", SubPath: "/", Handle: hello},
		},
	})

	return nil
}

func hello(w http.ResponseWriter, req *http.Request) ([]byte, error) {
	return []byte("hello, world\n"), nil
}
