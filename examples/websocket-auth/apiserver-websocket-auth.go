package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	osruntime "runtime"

	"github.com/yubo/apiserver/pkg/authentication"
	"github.com/yubo/apiserver/plugin/authn/token/tokentest"
	"github.com/yubo/apiserver/pkg/authentication/user"
	"github.com/yubo/apiserver/pkg/options"
	"github.com/yubo/apiserver/pkg/request"
	"github.com/yubo/apiserver/pkg/rest"
	"github.com/yubo/golib/proc"
	"github.com/yubo/golib/logs"
	"github.com/yubo/golib/util/runtime"
	"github.com/yubo/golib/util/wsstream"
	"golang.org/x/net/websocket"

	_ "github.com/yubo/apiserver/pkg/apiserver/register"
	_ "github.com/yubo/apiserver/pkg/authentication/register"
)

// This example shows the minimal code needed to get a restful.WebService working.

const (
	moduleName = "apiserver.hello"
	fakeUser   = "test"
	fakeToken  = "1234"
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

	osruntime.GOMAXPROCS(2)
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

func init() {
	proc.RegisterHooks(hookOps)

	authentication.RegisterTokenAuthn(&tokentest.TokenAuthenticator{
		Tokens: map[string]*user.DefaultInfo{
			fakeToken: &user.DefaultInfo{Name: fakeUser},
		},
	})

}
