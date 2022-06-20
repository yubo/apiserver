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

	server "github.com/yubo/apiserver/pkg/server/module"
	_ "github.com/yubo/apiserver/pkg/server/register"
)

const (
	moduleName = "hello.examples"
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
		Path:               "/hello",
		GoRestfulContainer: http,
		Routes: []rest.WsRoute{
			{Method: "GET", SubPath: "/", Handle: hello},
			{Method: "GET", SubPath: "/array", Handle: helloArray},
			{Method: "GET", SubPath: "/map", Handle: helloMap},
		},
	})
}

func hello(w http.ResponseWriter, req *http.Request) ([]byte, error) {
	return []byte("hello, world\n"), nil
}

func helloArray(w http.ResponseWriter, req *http.Request, _ *rest.NonParam, s *[]string) (string, error) {
	return fmt.Sprintf("%s", *s), nil
}

func helloMap(w http.ResponseWriter, req *http.Request, _ *rest.NonParam, m *map[string]string) (string, error) {
	return fmt.Sprintf("%s", *m), nil
}
