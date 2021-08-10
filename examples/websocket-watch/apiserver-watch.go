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
	"github.com/yubo/apiserver/staging/watch"
	"github.com/yubo/golib/proc"
	"github.com/yubo/golib/staging/logs"

	_ "github.com/yubo/apiserver/pkg/apiserver/register"
)

// This example shows the minimal code needed to get a restful.WebService working.
//
// GET http://localhost:8080/hello

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
		ticker := time.NewTicker(time.Second / 2)
		defer ticker.Stop()
		for {
			t := <-ticker.C
			if watcher.IsStopped() {
				return
			}
			watcher.Add(t)
		}
	}()

	return handlers.ServeWatch(watcher, req, w, 0)
}

func init() {
	proc.RegisterHooks(hookOps)
}
