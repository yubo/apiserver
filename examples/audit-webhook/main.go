package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/yubo/apiserver/pkg/options"
	"github.com/yubo/apiserver/pkg/rest"
	"github.com/yubo/golib/cli"
	"github.com/yubo/golib/proc"

	// http
	server "github.com/yubo/apiserver/pkg/server/module"
	_ "github.com/yubo/apiserver/pkg/server/register"

	// audit
	_ "github.com/yubo/apiserver/pkg/audit/register"
)

const (
	moduleName = "webhook.audit.examples"
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
		Path:               "/audit",
		GoRestfulContainer: http,
		Routes: []rest.WsRoute{
			{Method: "POST", SubPath: "/webhook", Handle: echo},
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
