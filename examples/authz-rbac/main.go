package main

import (
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
	_ "github.com/yubo/apiserver/plugin/authorizer/rbac/register"
)

// go run ./apiserver-authorization.go --token-auth-file=./tokens.cvs --authorization-mode=RBAC --rbac-provider=file --rbac-config-path=./testdata
// curl -X POST http://localhost:8080/api/v1/namespaces/test/users -H "Authorization: Bearer token-admin"

const (
	moduleName = "example.rbac.authz"
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
