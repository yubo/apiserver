package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	osruntime "runtime"
	"time"

	"github.com/yubo/apiserver/pkg/handlers"
	"github.com/yubo/apiserver/pkg/options"
	"github.com/yubo/apiserver/pkg/rest"
	"github.com/yubo/apiserver/pkg/watch"
	"github.com/yubo/golib/proc"
	"github.com/yubo/golib/logs"
	"k8s.io/klog/v2"

	_ "github.com/yubo/apiserver/pkg/apiserver/register"
)

// This example shows the minimal code needed to get a restful.WebService working.
//
// curl -X GET http://localhost:8080/hello
//
// go run ./apiserver-watch.go --request-timeout=10

const (
	moduleName = "apiserver.hello"
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
			watcher.Add(t)
		}
	}()

	err := handlers.ServeWatch(watcher, req, w, 0)
	klog.V(10).Infof("exit with err %v", err)
	return err
}

func init() {
	proc.RegisterHooks(hookOps)
}
