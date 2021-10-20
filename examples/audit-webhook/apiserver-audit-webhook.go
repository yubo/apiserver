package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/yubo/apiserver/pkg/options"
	"github.com/yubo/apiserver/pkg/rest"
	"github.com/yubo/golib/logs"
	"github.com/yubo/golib/proc"

	// http
	_ "github.com/yubo/apiserver/pkg/apiserver/register"

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
			{Method: "POST", SubPath: "/webhook/audit", Handle: echo},
			{Method: "POST", SubPath: "/hello", Handle: hw},
		},
	})
}

func echo(w http.ResponseWriter, req *http.Request) {
	body, _ := ioutil.ReadAll(req.Body)
	fmt.Printf("%s\n", string(body))
}

func hw(w http.ResponseWriter, req *http.Request) (string, error) {
	return "hello, world", nil
}

func init() {
	proc.RegisterHooks(hookOps)
}
