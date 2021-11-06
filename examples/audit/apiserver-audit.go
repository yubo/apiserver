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
	_ "github.com/yubo/apiserver/pkg/server/register"

	// audit
	_ "github.com/yubo/apiserver/pkg/audit/register"
)

// go run ./apiserver-audit.go
//

const (
	moduleName = "audit.example.apiserver"
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
		Path:               "/",
		GoRestfulContainer: http,
		Routes: []rest.WsRoute{
			{Method: "GET", SubPath: "/static/hw", Handle: hw},  // none
			{Method: "POST", SubPath: "/api/users", Handle: hw}, // RequestResponse
			{Method: "GET", SubPath: "/api/tokens", Handle: hw}, // metadata
		},
	})
}

func hw(w http.ResponseWriter, req *http.Request) (string, error) {
	return "hello, world", nil
}

func init() {
	proc.RegisterHooks(hookOps)
}
