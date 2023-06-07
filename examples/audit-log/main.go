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
		Path:               "/api",
		GoRestfulContainer: srv,
		Routes: []rest.WsRoute{
			{Method: "POST", SubPath: "/users", Handle: hw}, // RequestResponse
			{Method: "GET", SubPath: "/tokens", Handle: hw}, // metadata
		},
	})
	rest.WsRouteBuild(&rest.WsOption{
		Path:               "/static",
		GoRestfulContainer: srv,
		Routes: []rest.WsRoute{
			{Method: "GET", SubPath: "/hw", Handle: hw}, // none
		},
	})

	return nil
}

func hw(w http.ResponseWriter, req *http.Request) ([]byte, error) {
	return []byte(req.URL.Path + ": hello, world\n"), nil
}
