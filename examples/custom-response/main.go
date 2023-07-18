package main

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/yubo/apiserver/components/cli"
	"github.com/yubo/apiserver/components/dbus"
	"github.com/yubo/apiserver/pkg/proc"
	"github.com/yubo/apiserver/pkg/server"
	"github.com/yubo/apiserver/pkg/tracing"
	"github.com/yubo/apiserver/plugin/responsewriter/umi"
	"github.com/yubo/golib/util"
	"go.opentelemetry.io/otel/attribute"

	_ "github.com/yubo/apiserver/pkg/server/register"
)

// This example shows the minimal code needed to get a restful.WebService working.
// Open in browser http://localhost:8080/swagger

func main() {
	command := proc.NewRootCmd(proc.WithoutHTTPS(), proc.WithRun(start))
	code := cli.Run(command)
	os.Exit(code)
}

func start(ctx context.Context) error {
	srv, err := dbus.GetAPIServer()
	if err != nil {
		return err
	}
	server.SwaggerTagRegister("user", "user Api - swagger api sample")
	server.WsRouteBuild(&server.WsOption{
		Path:   "/api/v1/users",
		Tags:   []string{"user"},
		Server: srv,
		Routes: []server.WsRoute{{
			Method: "GET", SubPath: "/{name}",
			Desc:   "get user",
			Handle: getUser,
		}},
	})

	server.WsRouteBuild(&server.WsOption{
		Path:       "/api/v2/users",
		Tags:       []string{"user"},
		Server:     srv,
		RespWriter: umi.RespWriter,
		Routes: []server.WsRoute{{
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
	_, span := tracing.Start(req.Context(), "getUser", attribute.String("name", in.Name))
	defer span.End(100 * time.Millisecond)
	return &User{Name: in.Name, Phone: util.String("12345")}, nil
}
