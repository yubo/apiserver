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

// This example shows the minimal code needed to get a restful.WebService working.
//
// GET http://localhost:8080/hello

const (
	moduleName = "example.list.request"
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

type User struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type ListInput struct {
	rest.PageParams
}

type ListOutput struct {
	Total int     `json:"total"`
	List  []*User `json:"list"`
}

func installWs(http rest.GoRestfulContainer) {
	rest.WsRouteBuild(&rest.WsOption{
		Path:               "/users",
		GoRestfulContainer: http,
		Routes: []rest.WsRoute{
			{Method: "GET", SubPath: "/", Handle: list, Desc: "list users"},
		},
	})
}

func list(w http.ResponseWriter, req *http.Request, in *ListInput) (*ListOutput, error) {
	offset, limit := in.OffsetLimit()
	out := &ListOutput{Total: 1000}
	for i := 0; i < limit; i++ {
		out.List = append(out.List, &User{
			Name:        fmt.Sprintf("name-%03d", i+offset),
			Description: fmt.Sprintf("description for name-%03d", i+offset),
		})
	}
	return out, nil
}
