package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/yubo/apiserver/pkg/options"
	"github.com/yubo/apiserver/pkg/request"
	"github.com/yubo/apiserver/pkg/rest"
	"github.com/yubo/golib/logs"
	"github.com/yubo/golib/proc"

	// http
	_ "github.com/yubo/apiserver/pkg/server/register"

	// authn
	_ "github.com/yubo/apiserver/pkg/authentication/register"
	"github.com/yubo/apiserver/pkg/authentication/user"
	_ "github.com/yubo/apiserver/plugin/authenticator/token/tokenfile/register"
)

// go run ./apiserver-authentication.go --token-auth-file=./tokens.cvs
//
// This example shows the minimal code needed to get a restful.WebService working.
//
// curl -H 'Content-Type:application/json' -H 'Authorization: bearer token-777' http://localhost:8080/hello
// {
//   "Name": "user3",
//   "UID": "uid3",
//   "Groups": [
//     "group1",
//     "group2",
//     "system:authenticated"
//   ],
//   "Extra": null
// }

const (
	moduleName = "authn.example.apiserver"
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

	if err := proc.NewRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
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
			{Method: "GET", SubPath: "/", Handle: hw},
		},
	})
}

func hw(w http.ResponseWriter, req *http.Request) (*user.DefaultInfo, error) {
	u, ok := request.UserFrom(req.Context())
	if !ok {
		return nil, fmt.Errorf("unable to get user info")
	}
	return &user.DefaultInfo{
		Name:   u.GetName(),
		UID:    u.GetUID(),
		Groups: u.GetGroups(),
		Extra:  u.GetExtra(),
	}, nil
}

func init() {
	proc.RegisterHooks(hookOps)
}
