package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/yubo/apiserver/pkg/options"
	"github.com/yubo/apiserver/pkg/rest"
	"github.com/yubo/golib/logs"
	"github.com/yubo/golib/proc"

	// http
	_ "github.com/yubo/apiserver/pkg/apiserver/register"

	// authn
	_ "github.com/yubo/apiserver/pkg/authentication/register"
	_ "github.com/yubo/apiserver/plugin/authn/token/tokenfile/register"

	// authz
	_ "github.com/yubo/apiserver/plugin/authz/rbac/register"
	_ "github.com/yubo/apiserver/pkg/authorization/register"
)

// go run ./apiserver-authorization.go --token-auth-file=./tokens.cvs --authorization-mode=RBAC --rbac-provider=file --rbac-config-path=./testdata
// curl -X POST http://localhost:8080/api/v1/namespaces/test/users -H "Authorization: Bearer token-admin"

const (
	moduleName = "apiserver.authentication"
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
	logs.InitLogs()
	defer logs.FlushLogs()

	if err := proc.NewRootCmd(context.Background()).Execute(); err != nil {
		os.Exit(1)
	}
}

func start(ctx context.Context) error {
	http, ok := options.ApiServerFrom(ctx)
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

func init() {
	proc.RegisterHooks(hookOps)
}
