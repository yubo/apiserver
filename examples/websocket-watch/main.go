package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/yubo/apiserver/components/cli"
	"github.com/yubo/apiserver/pkg/handlers"
	"github.com/yubo/apiserver/pkg/proc"
	v1 "github.com/yubo/apiserver/pkg/proc/api/v1"
	"github.com/yubo/apiserver/pkg/proc/options"
	"github.com/yubo/apiserver/pkg/rest"
	"github.com/yubo/golib/watch"
	"k8s.io/klog/v2"

	server "github.com/yubo/apiserver/pkg/server/module"
	_ "github.com/yubo/apiserver/pkg/server/register"
)

// This example shows the minimal code needed to get a restful.WebService working.
//
// curl -X GET http://localhost:8080/hello
//
// go run ./apiserver-watch.go --request-timeout=10

const (
	moduleName = "example.watch.websocket"
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
			{Method: "GET", SubPath: "/", Handle: watchHandle},
		},
	})
}

func watchHandle(w http.ResponseWriter, req *http.Request) error {
	watcher := watch.NewFakeWithChanSize(2, false)

	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			t := <-ticker.C
			if watcher.IsStopped() {
				return
			}
			watcher.Add(t.String())
		}
	}()

	err := handlers.ServeWatch(watcher, req, w, 0)
	klog.V(10).Infof("exit with err %v", err)
	return err
}
