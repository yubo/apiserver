package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/yubo/apiserver/pkg/options"
	"github.com/yubo/apiserver/pkg/rest"
	"github.com/yubo/golib/proc"
	"github.com/yubo/golib/logs"

	// http
	_ "github.com/yubo/apiserver/pkg/apiserver/register"

	// authn
	_ "github.com/yubo/apiserver/pkg/authentication/register"
	_ "github.com/yubo/apiserver/pkg/authentication/token/tokenfile/register"

	// authz
	_ "github.com/yubo/apiserver/pkg/authorization/abac/register"
	_ "github.com/yubo/apiserver/pkg/authorization/register"
)

// go run ./apiserver-authorization.go --token-auth-file=./tokens.cvs --authorization-mode=ABAC  --authorization-policy-file=./abac.json
//
// This example shows the minimal code needed to get a restful.WebService working.

// curl -XGET -H 'Authorization: bearer token-777' http://localhost:8080/ro -I
// HTTP/1.1 200 OK
// Cache-Control: no-cache, private
// Date: Tue, 27 Jul 2021 11:33:54 GMT
// Content-Length: 0

// curl -XGET  -Ss -i http://localhost:8080/ro
// HTTP/1.1 403 Forbidden
// Cache-Control: no-cache, private
// Content-Type: application/json
// X-Content-Type-Options: nosniff
// Date: Tue, 27 Jul 2021 13:41:45 GMT
// Content-Length: 239
//
// {
//   "kind": "Status",
//   "apiVersion": "v1",
//   "metadata": {},
//   "status": "Failure",
//   "message": "forbidden: User \"system:anonymous\" cannot get path \"/ro\": No policy matched.",
//   "reason": "Forbidden",
//   "details": {},
//   "code": 403
// }

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
		Path:               "/",
		GoRestfulContainer: http,
		Routes: []rest.WsRoute{
			{Method: "GET", SubPath: "/ro", Handle: handle},
			{Method: "POST", SubPath: "/rw", Handle: handle},
			{Method: "GET", SubPath: "/unauthenticated", Handle: handle},
		},
	})
}

func handle(w http.ResponseWriter, req *http.Request) error {
	return nil
}

func init() {
	proc.RegisterHooks(hookOps)
}
