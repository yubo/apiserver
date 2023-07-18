package main

import (
	"context"
	"net/http"
	"os"

	"github.com/yubo/apiserver/components/cli"
	"github.com/yubo/apiserver/components/dbus"
	"github.com/yubo/apiserver/pkg/proc"
	"github.com/yubo/apiserver/pkg/server"

	_ "github.com/yubo/apiserver/pkg/server/register"
)

// go run ./apiserver-authorization.go --token-auth-file=./tokens.cvs --authorization-mode=RBAC --rbac-provider=file --rbac-config-path=./testdata
// curl -X POST http://localhost:8080/api/v1/namespaces/test/users -H "Authorization: Bearer token-admin"

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
		Path:     "/api/v1",
		Server:   srv,
		Consumes: []string{server.MIME_ALL},
		Routes: []server.WsRoute{
			// with clusterRole & ClusterRoleBinding
			{Method: "GET", SubPath: "/users", Handle: handle},
			{Method: "POST", SubPath: "/users", Handle: handle},
			{Method: "GET", SubPath: "/metrics", Handle: handle},

			// with role & roleBinding (test namespace)
			{Method: "GET", SubPath: "/namespaces/test/users", Handle: handle},
			{Method: "POST", SubPath: "/namespaces/test/users", Handle: handle},
			{Method: "GET", SubPath: "/namespaces/test/metrics", Handle: handle},
		},
	})

	return nil
}

func handle(w http.ResponseWriter, req *http.Request) error {
	return nil
}
