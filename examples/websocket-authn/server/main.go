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
	v1 "github.com/yubo/apiserver/pkg/proc/api/v1"
	"github.com/yubo/apiserver/pkg/proc/options"
	"github.com/yubo/apiserver/pkg/request"
	"github.com/yubo/apiserver/pkg/rest"
	"github.com/yubo/apiserver/plugin/authenticator/token/tokentest"
	"github.com/yubo/golib/stream/wsstream"
	"github.com/yubo/golib/util/runtime"
	"golang.org/x/net/websocket"

	_ "github.com/yubo/apiserver/pkg/authentication/register"
	server "github.com/yubo/apiserver/pkg/server/module"
	_ "github.com/yubo/apiserver/pkg/server/register"
)

// This example shows the minimal code needed to get a restful.WebService working.

const (
	moduleName = "example.authn.websocket"
	fakeUser   = "test"
	fakeToken  = "1234"
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
	registerAuthn()

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
			{Method: "GET", SubPath: "/", Handle: wsHandle},
		},
	})
}

func wsHandle(w http.ResponseWriter, req *http.Request) error {
	if !wsstream.IsWebSocketRequest(req) {
		return fmt.Errorf("not a websocket request")
	}

	w.Header().Set("Content-Type", rest.MIME_TXT)
	websocket.Handler(_wsHandle).ServeHTTP(w, req)
	return nil
}

func _wsHandle(ws *websocket.Conn) {
	defer ws.Close()

	go func() {
		defer runtime.HandleCrash()
		// This blocks until the connection is closed.
		// Client should not send anything.
		wsstream.IgnoreReceives(ws, 0)
	}()

	u, ok := request.UserFrom(ws.Request().Context())
	if !ok {
		websocket.Message.Send(ws, "unable get userinfo")
		return
	}

	websocket.Message.Send(ws, fmt.Sprintf("username: %s, groups: %s",
		u.GetName(), u.GetGroups()))
}

func registerAuthn() {
	authentication.RegisterTokenAuthn(func(context.Context) (authenticator.Token, error) {
		return &tokentest.TokenAuthenticator{
			Tokens: map[string]*user.DefaultInfo{
				fakeToken: {Name: fakeUser},
			},
		}, nil
	})
}
