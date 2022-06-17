package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/yubo/apiserver/pkg/authentication"
	"github.com/yubo/apiserver/pkg/authentication/authenticator"
	"github.com/yubo/apiserver/pkg/authentication/user"
	"github.com/yubo/apiserver/pkg/options"
	"github.com/yubo/apiserver/pkg/request"
	"github.com/yubo/apiserver/pkg/rest"
	"github.com/yubo/golib/cli"
	"github.com/yubo/golib/proc"

	// http
	server "github.com/yubo/apiserver/pkg/server/module"
	_ "github.com/yubo/apiserver/pkg/server/register"

	// authn
	_ "github.com/yubo/apiserver/pkg/authentication/register"
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
	authentication.RegisterTokenAuthn(func(_ context.Context) (authenticator.Token, error) {
		return &TokenAuthenticator{}, nil
	})

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
			{Method: "GET", SubPath: "/", Handle: hw},
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

type TokenAuthenticator struct{}

var (
	_token = "123"
	_user  = &user.DefaultInfo{
		Name: "system",
	}
)

func (a *TokenAuthenticator) AuthenticateToken(ctx context.Context, value string) (*authenticator.Response, bool, error) {
	if value == _token {
		return &authenticator.Response{User: _user}, true, nil
	}

	return nil, false, nil
}

func (a *TokenAuthenticator) Name() string {
	return "custom token authenticator"
}

func (a *TokenAuthenticator) Priority() int {
	return authenticator.PRI_TOKEN_CUSTOM
}

func (a *TokenAuthenticator) Available() bool {
	return true
}
