package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/yubo/apiserver/components/cli"
	"github.com/yubo/apiserver/components/dbus"
	"github.com/yubo/apiserver/pkg/authentication/user"
	"github.com/yubo/apiserver/pkg/proc"
	v1 "github.com/yubo/apiserver/pkg/proc/api/v1"
	"github.com/yubo/apiserver/pkg/request"
	"github.com/yubo/apiserver/pkg/server"

	_ "github.com/yubo/apiserver/pkg/server/register"
)

// This example shows the minimal code needed to get a restful.WebService working.

const (
	moduleName = "x509.authn.examples"
)

var (
	hookOps = []v1.HookOps{{
		Hook:     start,
		Owner:    moduleName,
		HookNum:  v1.ACTION_START,
		Priority: v1.PRI_MODULE,
	}}
)

func main() {
	command := proc.NewRootCmd(proc.WithHooks(hookOps...))
	code := cli.Run(command)
	os.Exit(code)
}

func start(ctx context.Context) error {
	srv, err := dbus.GetAPIServer()
	if err != nil {
		return err
	}
	server.WsRouteBuild(&server.WsOption{
		Path:   "/inc",
		Server: srv,
		Routes: []server.WsRoute{
			{Method: "POST", SubPath: "/", Handle: inc},
		},
	})
	return nil
}

type Input struct {
	X int
}

type Output struct {
	X    int
	User user.DefaultInfo
}

func inc(w http.ResponseWriter, req *http.Request, input *Input) (*Output, error) {
	u, ok := request.UserFrom(req.Context())
	if !ok {
		return nil, fmt.Errorf("unable to get user info")
	}

	return &Output{
		X: input.X + 1,
		User: user.DefaultInfo{
			Name:   u.GetName(),
			UID:    u.GetUID(),
			Groups: u.GetGroups(),
			Extra:  u.GetExtra(),
		},
	}, nil
}
