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
//
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

type Item struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type ListInput struct {
	rest.Pagination
}

type ListOutput struct {
	Total int     `json:"total"`
	List  []*Item `json:"list"`
}

func installWs(http rest.GoRestfulContainer) {
	rest.WsRouteBuild(&rest.WsOption{
		Path:               "/",
		GoRestfulContainer: http,
		Routes: []rest.WsRoute{
			{Method: "GET", SubPath: "/", Handle: list, Desc: "list itmes"},
		},
	})
}

func list(w http.ResponseWriter, req *http.Request, in *ListInput) (*ListOutput, error) {
	offset, limit := in.OffsetLimit()
	out := &ListOutput{Total: 1000}
	for i := 0; i < limit; i++ {
		out.List = append(out.List, &Item{
			Name:        fmt.Sprintf("name-%03d", i+offset),
			Description: fmt.Sprintf("description for name-%03d", i+offset),
		})
	}
	return out, nil
}

func init() {
	proc.RegisterHooks(hookOps)
}
