package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/yubo/apiserver/pkg/options"
	"github.com/yubo/apiserver/pkg/rest"
	servermodule "github.com/yubo/apiserver/pkg/server/module"
	"github.com/yubo/apiserver/pkg/streaming"
	"github.com/yubo/apiserver/pkg/streaming/api"
	"github.com/yubo/apiserver/pkg/streaming/portforward"
	"github.com/yubo/apiserver/pkg/streaming/providers/dockershim"
	remotecommandserver "github.com/yubo/apiserver/pkg/streaming/remotecommand"
	"github.com/yubo/golib/cli"
	"github.com/yubo/golib/proc"
	"k8s.io/klog/v2"

	_ "github.com/yubo/apiserver/pkg/server/register"
)

// This example shows the minimal code needed to get a restful.WebService working.
//
// curl -X GET http://localhost:8080/hello
//
// go run ./apiserver-watch.go --request-timeout=10

const (
	moduleName = "example.container.remotecommand"
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
		provider: dockershim.NewProvider("unix:///var/run/docker.sock", 2*time.Minute, time.Minute),
	}
)

type server struct {
	config   streaming.Config
	provider streaming.Provider
}

func main() {
	command := proc.NewRootCmd(servermodule.WithoutTLS(), proc.WithHooks(hookOps...))
	code := cli.Run(command)
	os.Exit(code)
}

func start(ctx context.Context) error {
	http, ok := options.APIServerFrom(ctx)
	if !ok {
		return fmt.Errorf("unable to get http server from the context")
	}

	_server.installWs(http)
	return nil
}

func (p *server) installWs(http rest.GoRestfulContainer) {
	rest.WsRouteBuild(&rest.WsOption{
		Path:               "/remotecommand",
		GoRestfulContainer: http,
		Routes: []rest.WsRoute{
			{Method: "POST", SubPath: "/exec", Handle: p.exec},
			{Method: "POST", SubPath: "/attach", Handle: p.attach},
			{Method: "POST", SubPath: "/portforward", Handle: p.portForward},
		},
	})
}

func (p *server) exec(w http.ResponseWriter, req *http.Request, in *api.ExecRequest) error {
	klog.Infof("entering exec %#v", in)
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
