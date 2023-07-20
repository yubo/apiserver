package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/yubo/apiserver/components/cli"
	"github.com/yubo/apiserver/components/dbus"
	"github.com/yubo/apiserver/pkg/authentication/authenticator"
	"github.com/yubo/apiserver/pkg/authentication/user"
	"github.com/yubo/apiserver/pkg/proc"
	"github.com/yubo/apiserver/pkg/request"
	"github.com/yubo/apiserver/pkg/server"
	authn "github.com/yubo/apiserver/pkg/server/authenticator"
	"github.com/yubo/apiserver/plugin/authenticator/token/tokentest"
	"github.com/yubo/golib/stream/wsstream"
	"github.com/yubo/golib/util/runtime"
	"golang.org/x/net/websocket"

	_ "github.com/yubo/apiserver/pkg/server/register"
)

// This example shows the minimal code needed to get a restful.WebService working.

const (
	moduleName = "example.authn.websocket"
	fakeUser   = "test"
	fakeToken  = "1234"
)

func main() {

	cmd := proc.NewRootCmd(
		proc.WithoutHTTPS(),
		proc.WithRun(start),
		proc.WithRegisterAuth(registerAuthn),
	)
	code := cli.Run(cmd)
	os.Exit(code)
}

func start(ctx context.Context) error {
	srv, err := dbus.GetAPIServer()
	if err != nil {
		return err
	}

	server.WsRouteBuild(&server.WsOption{
		Path:   "/hello",
		Server: srv,
		Routes: []server.WsRoute{
			{Method: "GET", SubPath: "/", Handle: wsHandle},
		},
	})

	return nil
}

func wsHandle(w http.ResponseWriter, req *http.Request) error {
	if !wsstream.IsWebSocketRequest(req) {
		return fmt.Errorf("not a websocket request")
	}

	w.Header().Set("Content-Type", server.MIME_TXT)
	websocket.Handler(wsHandle2).ServeHTTP(w, req)
	return nil
}

func wsHandle2(ws *websocket.Conn) {
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

func registerAuthn(ctx context.Context) error {
	authn.RegisterTokenAuthn(func(context.Context) (authenticator.Token, error) {
		return &tokentest.TokenAuthenticator{
			Tokens: map[string]*user.DefaultInfo{
				fakeToken: {Name: fakeUser},
			},
		}, nil
	})

	return nil
}
