package main

import (
	// set default config, must before the other modules
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

	// authn
	_ "github.com/yubo/apiserver/pkg/authentication/register"
	_ "github.com/yubo/apiserver/pkg/authentication/token/tokenfile/register"

	// authz
	_ "github.com/yubo/apiserver/pkg/authorization/register"
	_ "github.com/yubo/apiserver/plugin/authorizer/abac/register"
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
			{Method: "GET", SubPath: "/ro", Handle: handle},
			{Method: "GET", SubPath: "/rw", Handle: handle},
			{Method: "GET", SubPath: "/unauthenticated", Handle: handle},
		},
	})
	return nil
}

func handle(w http.ResponseWriter, req *http.Request) ([]byte, error) {
	return []byte("hello\n"), nil
}
