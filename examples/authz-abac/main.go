package main

import (
	// set default config, must before the other modules
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/yubo/apiserver/components/cli"
	"github.com/yubo/apiserver/pkg/proc"
	v1 "github.com/yubo/apiserver/pkg/proc/api/v1"
	"github.com/yubo/apiserver/pkg/proc/options"
	"github.com/yubo/apiserver/pkg/rest"

	// http
	server "github.com/yubo/apiserver/pkg/server/module"
	_ "github.com/yubo/apiserver/pkg/server/register"

	// authn
	_ "github.com/yubo/apiserver/pkg/authentication/register"
	_ "github.com/yubo/apiserver/plugin/authenticator/token/tokenfile/register"

	// authz
	_ "github.com/yubo/apiserver/pkg/authorization/register"
	_ "github.com/yubo/apiserver/plugin/authorizer/abac/register"
)

const (
	moduleName = "example.abac.authz"
)

var (
	hookOps = []v1.HookOps{{
		Hook:     start,
		Owner:    moduleName,
		HookNum:  v1.ACTION_START,
		Priority: v1.PRI_MODULE,
	}}
)

func main() {
	command := proc.NewRootCmd(server.WithoutTLS(), proc.WithHooks(hookOps...))
	code := cli.Run(command)
	os.Exit(code)
}

func start(ctx context.Context) error {
	http, ok := options.APIServerFrom(ctx)
	if !ok {
		return fmt.Errorf("unable to get http server from the context")
	}

	installWs(http)
	return nil
}

func installWs(http rest.GoRestfulContainer) {
	rest.WsRouteBuild(&rest.WsOption{
		Path:               "/hello",
		GoRestfulContainer: http,
		Routes: []rest.WsRoute{
			{Method: "GET", SubPath: "/ro", Handle: handle},
			{Method: "GET", SubPath: "/rw", Handle: handle},
			{Method: "GET", SubPath: "/unauthenticated", Handle: handle},
		},
	})
}

func handle(w http.ResponseWriter, req *http.Request) ([]byte, error) {
	return []byte("hello\n"), nil
}
