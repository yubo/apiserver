package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/yubo/apiserver/components/cli"
	"github.com/yubo/apiserver/pkg/authentication"
	"github.com/yubo/apiserver/pkg/authentication/authenticator"
	"github.com/yubo/apiserver/pkg/authentication/user"
	"github.com/yubo/apiserver/pkg/proc"
	"github.com/yubo/apiserver/pkg/proc/options"
	"github.com/yubo/apiserver/pkg/request"
	"github.com/yubo/apiserver/pkg/rest"

	// http
	server "github.com/yubo/apiserver/pkg/server/module"
	_ "github.com/yubo/apiserver/pkg/server/register"

	// authz
	_ "github.com/yubo/apiserver/pkg/authorization/register"
	_ "github.com/yubo/apiserver/plugin/authorizer/abac/register"

	// authn
	_ "github.com/yubo/apiserver/pkg/authentication/register"
)

func main() {
	authentication.RegisterTokenAuthn(func(_ context.Context) (authenticator.Token, error) {
		return &TokenAuthenticator{}, nil
	})

	command := proc.NewRootCmd(server.WithoutTLS(), proc.WithRun(start))
	code := cli.Run(command)
	os.Exit(code)
}

func start(ctx context.Context) error {
	srv, ok := options.APIServerFrom(ctx)
	if !ok {
		return fmt.Errorf("unable to get http server from the context")
	}
	rest.WsRouteBuild(&rest.WsOption{
		Path:               "/hello",
		GoRestfulContainer: srv,
		Routes: []rest.WsRoute{
			{Method: "GET", SubPath: "/", Handle: hw, Scope: "auth"},
		},
	})

	return nil
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
		Name:   "steve",
		Groups: []string{"dev"},
	}
)

func (a *TokenAuthenticator) AuthenticateToken(ctx context.Context, value string) (*authenticator.Response, bool, error) {
	if value == _token {
		return &authenticator.Response{User: _user}, true, nil
	}

	return nil, false, nil
}
