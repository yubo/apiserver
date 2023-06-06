package main

import (
	"context"
	"net/http"
	"os"

	"github.com/yubo/apiserver/components/cli"
	"github.com/yubo/apiserver/components/dbus"
	"github.com/yubo/apiserver/pkg/proc"
	"github.com/yubo/apiserver/pkg/rest"

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
	rest.WsRouteBuild(&rest.WsOption{
		Path:               "/hello",
		GoRestfulContainer: srv,
		Routes: []rest.WsRoute{
			{Method: "GET", SubPath: "/", Handle: hello},
		},
	})

	return nil
}

func hello(w http.ResponseWriter, req *http.Request) ([]byte, error) {
	return []byte("hello, world\n"), nil
}
