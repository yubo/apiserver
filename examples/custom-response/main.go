package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/yubo/apiserver/components/cli"
	"github.com/yubo/apiserver/pkg/proc"
	"github.com/yubo/apiserver/pkg/proc/options"
	"github.com/yubo/apiserver/pkg/rest"
	"github.com/yubo/apiserver/plugin/responsewriter/umi"
	"github.com/yubo/golib/util"

	server "github.com/yubo/apiserver/pkg/server/module"
	_ "github.com/yubo/apiserver/pkg/server/register"
)

// This example shows the minimal code needed to get a restful.WebService working.
// Open in browser http://localhost:8080/swagger

func main() {
	command := proc.NewRootCmd(server.WithoutTLS(), proc.WithRun(start))
	code := cli.Run(command)
	os.Exit(code)
}

func start(ctx context.Context) error {
	srv, ok := options.APIServerFrom(ctx)
	if !ok {
		return fmt.Errorf("unable to get http server from the context")
	}
	rest.SwaggerTagRegister("user", "user Api - swagger api sample")
	rest.WsRouteBuild(&rest.WsOption{
		Path:               "/api/v1/users",
		Tags:               []string{"user"},
		GoRestfulContainer: srv,
		Routes: []rest.WsRoute{{
			Method: "GET", SubPath: "/{name}",
			Desc:   "get user",
			Handle: getUser,
		}},
	})

	rest.WsRouteBuild(&rest.WsOption{
		Path:               "/api/v2/users",
		Tags:               []string{"user"},
		GoRestfulContainer: srv,
		RespWriter:         umi.RespWriter,
		Routes: []rest.WsRoute{{
			Method: "GET", SubPath: "/{name}",
			Desc:   "get user",
			Handle: getUser,
		}},
	})

	return nil
}

type User struct {
	Name     string  `json:"name"`
	NickName *string `json:"nickName"`
	Phone    *string `json:"phone"`
}

type GetUserInput struct {
	Name string `param:"path" name:"name" description:"query user name or nick name"`
}

func getUser(w http.ResponseWriter, req *http.Request, in *GetUserInput) (*User, error) {
	return &User{Name: in.Name, Phone: util.String("12345")}, nil
}
