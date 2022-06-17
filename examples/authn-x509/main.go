package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/yubo/apiserver/pkg/options"
	"github.com/yubo/apiserver/pkg/request"
	"github.com/yubo/apiserver/pkg/rest"
	"github.com/yubo/golib/cli"
	"github.com/yubo/golib/proc"

	_ "github.com/yubo/apiserver/pkg/authentication/register"
	"github.com/yubo/apiserver/pkg/authentication/user"
	_ "github.com/yubo/apiserver/pkg/server/register"
	_ "github.com/yubo/apiserver/plugin/authenticator/x509/register"
)

// This example shows the minimal code needed to get a restful.WebService working.

const (
	moduleName = "x509.authn.examples"
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
	command := proc.NewRootCmd(proc.WithHooks(hookOps...))
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
		Path:               "/inc",
		GoRestfulContainer: http,
		Routes: []rest.WsRoute{
			{Method: "POST", SubPath: "/", Handle: inc},
		},
	})
}

type Input struct {
	X int
}

type Output struct {
	X    int
	User user.DefaultInfo
}

func inc(w http.ResponseWriter, req *http.Request, _ *rest.NonParam, input *Input) (*Output, error) {
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
