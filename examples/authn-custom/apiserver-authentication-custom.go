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
	"github.com/yubo/golib/proc"
	"github.com/yubo/golib/staging/logs"

	// http
	_ "github.com/yubo/apiserver/pkg/apiserver/register"

	// authn
	_ "github.com/yubo/apiserver/pkg/authentication/register"
)

// go run ./apiserver-authentication-custom.go
//
// $ curl -Ss -i http://localhost:8080/hello
// HTTP/1.1 200 OK
// Cache-Control: no-cache, private
// Content-Type: application/json
// Content-Length: 103
//
// {
//  "Name": "system:anonymous",
//  "UID": "",
//  "Groups": [
//   "system:unauthenticated"
//  ],
//  "Extra": null
// }
//
// $ curl -Ss  -H 'Authorization: bearer 123' http://localhost:8080/hello
// {
//  "Name": "system",
//  "UID": "",
//  "Groups": [
//   "system:authenticated"
//  ],
//  "Extra": null
// }

const (
	moduleName = "apiserver.authentication"
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

	proc.RegisterHooks(hookOps)
	authentication.RegisterTokenAuthn(&TokenAuthenticator{})

	if err := proc.NewRootCmd(context.Background()).Execute(); err != nil {
		os.Exit(1)
	}
}

func start(ctx context.Context) error {
	http, ok := options.ApiServerFrom(ctx)
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
