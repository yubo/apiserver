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

	// authn
	_ "github.com/yubo/apiserver/pkg/authentication/register"
	_ "github.com/yubo/apiserver/pkg/authentication/token/tokenfile/register"

	// authz
	_ "github.com/yubo/apiserver/pkg/authorization/register"
	_ "github.com/yubo/apiserver/plugin/authorizer/rbac/register"
)

// go run ./apiserver-authorization.go --token-auth-file=./tokens.cvs --authorization-mode=RBAC --rbac-provider=file --rbac-config-path=./testdata
// curl -X POST http://localhost:8080/api/v1/namespaces/test/users -H "Authorization: Bearer token-admin"

func main() {
	command := proc.NewRootCmd(server.WithoutTLS(), proc.WithRun(start))
	code := cli.Run(command)
	os.Exit(code)
}

func start(ctx context.Context) error {
	srv, err := dbus.GetAPIServer()
	if err != nil {
		return err
	}
	rest.WsRouteBuild(&rest.WsOption{
		Path:               "/api/v1",
		GoRestfulContainer: srv,
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

	return nil
}

func handle(w http.ResponseWriter, req *http.Request) error {
	return nil
}
