package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/yubo/apiserver/pkg/authentication/user"
	"github.com/yubo/apiserver/pkg/options"
	"github.com/yubo/apiserver/pkg/request"
	"github.com/yubo/apiserver/pkg/rest"
	"github.com/yubo/golib/cli"
	"github.com/yubo/golib/proc"

	// http
	_ "github.com/yubo/apiserver/pkg/server/register"

	// authz
	_ "github.com/yubo/apiserver/pkg/authorization/register"
	_ "github.com/yubo/apiserver/plugin/authorizer/abac/register"

	// authn
	_ "github.com/yubo/apiserver/pkg/authentication/register"
	"github.com/yubo/apiserver/plugin/authenticator/basic"
)

const (
	moduleName = "custom.authn.examples"
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
	basic.RegisterAuthn(&basicAuthenticator{})
	rest.ScopeRegister("auth", "auth description")

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
	rest.SwaggerTagRegister("demo", "demo Api - swagger api sample")
	rest.WsRouteBuild(&rest.WsOption{
		Path:               "/hello",
		GoRestfulContainer: http,
		Routes: []rest.WsRoute{
			{Method: "GET", SubPath: "/", Handle: hw, Scope: "auth"},
		},
	})
}

func hw(w http.ResponseWriter, req *http.Request) (*user.DefaultInfo, error) {
	u, ok := request.UserFrom(req.Context())
	if !ok {
		return nil, fmt.Errorf("unable to get user info")
	}
	return &user.DefaultInfo{
		Name:   u.GetName(),
		UID:    u.GetUID(),
		Groups: u.GetGroups(),
		Extra:  u.GetExtra(),
	}, nil
}

type basicAuthenticator struct{}

var (
	_username = "steve"
	_password = "123"
	_user     = &user.DefaultInfo{
		Name:   "steve",
		Groups: []string{"dev"},
	}
)

func (a *basicAuthenticator) Authenticate(user, pwd string) user.Info {
	if user == _username && pwd == _password {
		return _user
	}

	return nil
}
