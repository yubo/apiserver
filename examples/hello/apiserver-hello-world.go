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

	_ "github.com/yubo/apiserver/pkg/server/register"
)

// This example shows the minimal code needed to get a restful.WebService working.
// go run ./apiserver-hello-world.go --secure-serving=false --insecure-serving
// GET http://localhost:8080/hello

const (
	moduleName = "apiserver.hello"
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

	proc.RegisterHooks(hookOps)

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
	return fmt.Sprintf("hello, %+v", s), nil
}

func helloMap(w http.ResponseWriter, req *http.Request, _ *rest.NonParam, m *map[string]string) (string, error) {
	return fmt.Sprintf("hello, %+v", m), nil
}
