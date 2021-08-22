package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	osruntime "runtime"

	"github.com/emicklei/go-restful"
	"github.com/yubo/apiserver/pkg/options"
	"github.com/yubo/apiserver/pkg/rest"
	"github.com/yubo/apiserver/pkg/streaming"
	"github.com/yubo/apiserver/pkg/streaming/api"
	"github.com/yubo/apiserver/pkg/streaming/portforward"
	"github.com/yubo/apiserver/pkg/streaming/providers/native"
	remotecommandserver "github.com/yubo/apiserver/pkg/streaming/remotecommand"
	"github.com/yubo/golib/logs"
	"github.com/yubo/golib/proc"
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
	_server = &server{
		config:   streaming.DefaultConfig,
		provider: native.NewProvider(),
	}
)

type server struct {
	config   streaming.Config
	provider streaming.Provider
}

func main() {
	logs.InitLogs()
	defer logs.FlushLogs()

	osruntime.GOMAXPROCS(2)

	restful.EnableTracing(true)

	if err := proc.NewRootCmd(context.Background()).Execute(); err != nil {
		os.Exit(1)
	}
}

func start(ctx context.Context) error {
	http, ok := options.ApiServerFrom(ctx)
	if !ok {
		return fmt.Errorf("unable to get http server from the context")
	}

	_server.installWs(http)
	return nil
}

func (p *server) installWs(http rest.GoRestfulContainer) {
	rest.WsRouteBuild(&rest.WsOption{
		Path:               "/",
		GoRestfulContainer: http,
		Routes: []rest.WsRoute{
			{Method: "POST", SubPath: "/exec", Handle: p.exec},
			{Method: "GET", SubPath: "/exec", Handle: p.exec},
			{Method: "POST", SubPath: "/attach", Handle: p.attach},
			{Method: "POST", SubPath: "/portforward", Handle: p.portForward},
		},
	})
}

func (p *server) exec(w http.ResponseWriter, req *http.Request, in *api.ExecRequest) error {
	klog.Info("entering exec")
	streamOpts := &remotecommandserver.Options{
		Stdin:  in.Stdin,
		Stdout: in.Stdout,
		Stderr: in.Stderr,
		TTY:    in.Tty,
	}

	remotecommandserver.ServeExec(
		w,
		req,
		p.provider,
		"", // unused: podName
		"", // unusued: podUID
		in.ContainerId,
		in.Cmd,
		streamOpts,
		p.config.StreamIdleTimeout,
		p.config.StreamCreationTimeout,
		p.config.SupportedRemoteCommandProtocols)

	return nil

}

func (p *server) attach(w http.ResponseWriter, req *http.Request, in *api.AttachRequest) error {
	streamOpts := &remotecommandserver.Options{
		Stdin:  in.Stdin,
		Stdout: in.Stdout,
		Stderr: in.Stderr,
		TTY:    in.Tty,
	}
	remotecommandserver.ServeAttach(
		w,
		req,
		p.provider,
		"", // unused: podName
		"", // unusued: podUID
		in.ContainerId,
		streamOpts,
		p.config.StreamIdleTimeout,
		p.config.StreamCreationTimeout,
		p.config.SupportedRemoteCommandProtocols)
	return nil
}

func (p *server) portForward(w http.ResponseWriter, req *http.Request, in *api.PortForwardRequest) error {
	portForwardOptions, err := portforward.BuildV4Options(in.Port)
	if err != nil {
		return err
	}

	portforward.ServePortForward(
		w,
		req,
		p.provider,
		in.PodSandboxId,
		"", // unused: podUID
		portForwardOptions,
		p.config.StreamIdleTimeout,
		p.config.StreamCreationTimeout,
		p.config.SupportedPortForwardProtocols)
	return nil
}

func init() {
	proc.RegisterHooks(hookOps)
}
