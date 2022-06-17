package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/yubo/apiserver/pkg/options"
	"github.com/yubo/apiserver/pkg/rest"
	"github.com/yubo/golib/cli"
	"github.com/yubo/golib/proc"

	// http
	server "github.com/yubo/apiserver/pkg/server/module"
	_ "github.com/yubo/apiserver/pkg/server/register"

	// authn
	_ "github.com/yubo/apiserver/pkg/authentication/register"
	_ "github.com/yubo/apiserver/plugin/authenticator/token/tokenfile/register"

	// authz
	_ "github.com/yubo/apiserver/pkg/authorization/register"
	_ "github.com/yubo/apiserver/plugin/authorizer/rbac/register"
)

const (
	moduleName = "webhook.authz.examples"
)

var (
	hookOps = []proc.HookOps{{
		Hook:     start,
		Owner:    moduleName,
		HookNum:  proc.ACTION_START,
		Priority: proc.PRI_MODULE,
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
		Path:               "/api/v1",
		GoRestfulContainer: http,
		Routes: []rest.WsRoute{
			// with clusterRole & ClusterRoleBinding
			{Method: "GET", SubPath: "/users", Handle: handle},
			{Method: "POST", SubPath: "/users", Handle: handle},
			{Method: "GET", SubPath: "/status", Handle: handle},

			// with role & roleBinding (test namespace)
			{Method: "GET", SubPath: "/namespaces/test/users", Handle: handle},
			{Method: "POST", SubPath: "/namespaces/test/users", Handle: handle},
			{Method: "GET", SubPath: "/namespaces/test/status", Handle: handle},
		},
	})
}

func handle(w http.ResponseWriter, req *http.Request) error {
	return nil
}
