package main

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/yubo/apiserver/components/cli"
	"github.com/yubo/apiserver/components/dbus"
	"github.com/yubo/apiserver/pkg/handlers"
	"github.com/yubo/apiserver/pkg/proc"
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

func main() {
	command := proc.NewRootCmd(
		server.WithoutTLS(),
		proc.WithRun(start),
	)
	code := cli.Run(command)
	os.Exit(code)
}

func start(ctx context.Context) error {
	http, err := dbus.GetAPIServer()
	if err != nil {
		return err
	}

	installWs(http)
	return nil
}

func installWs(http rest.GoRestfulContainer) {
	server.WsRouteBuild(&server.WsOption{
		Path:               "/hello",
		GoRestfulContainer: http,
		Routes: []server.WsRoute{
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
