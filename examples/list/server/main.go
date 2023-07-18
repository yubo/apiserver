package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/yubo/apiserver/components/cli"
	"github.com/yubo/apiserver/components/dbus"
	"github.com/yubo/apiserver/pkg/proc"
	"github.com/yubo/apiserver/pkg/server"
	"github.com/yubo/golib/api"

	_ "github.com/yubo/apiserver/pkg/server/register"
)

// This example shows the minimal code needed to get a restful.WebService working.
//
// GET http://localhost:8080/hello

func main() {
	command := proc.NewRootCmd(proc.WithRun(start), proc.WithoutHTTPS())
	code := cli.Run(command)
	os.Exit(code)
}

func start(ctx context.Context) error {
	srv, err := dbus.GetAPIServer()
	if err != nil {
		return err
	}
	server.WsRouteBuild(&server.WsOption{
		Path:   "/users",
		Server: srv,
		Routes: []server.WsRoute{
			{Method: "GET", SubPath: "/", Handle: list, Desc: "list users"},
		},
	})

	return nil
}

type User struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type ListInput struct {
	api.PageParams
}

type ListOutput struct {
	Total int     `json:"total"`
	List  []*User `json:"list"`
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
