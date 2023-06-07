package main

import (
	"context"
	"net/http"
	"os"

	"github.com/yubo/apiserver/components/cli"
	"github.com/yubo/apiserver/components/dbus"
	"github.com/yubo/apiserver/pkg/proc"
	"github.com/yubo/apiserver/pkg/rest"
	servermodule "github.com/yubo/apiserver/pkg/server/module"
	"github.com/yubo/apiserver/pkg/streaming"
	"github.com/yubo/apiserver/pkg/streaming/portforward"
	"github.com/yubo/apiserver/pkg/streaming/providers/native"
	remotecommandserver "github.com/yubo/apiserver/pkg/streaming/remotecommand"
	"github.com/yubo/golib/api"
	"k8s.io/klog/v2"

	_ "github.com/yubo/apiserver/pkg/server/register"
)

// This example shows the minimal code needed to get a restful.WebService working.
//
// curl -X GET http://localhost:8080/hello
//
// go run ./apiserver-watch.go --request-timeout=10

type server struct {
	config   streaming.Config
	provider streaming.Provider
}

func main() {
	cmd := proc.NewRootCmd(servermodule.WithoutTLS(), proc.WithRun(start))
	code := cli.Run(cmd)
	os.Exit(code)
}

func start(ctx context.Context) error {
	http, err := dbus.GetAPIServer()
	if err != nil {
		return err
	}

	recorderProvider, err := native.NewFileRecorderProvider("/tmp")
	if err != nil {
		return err
	}

	srv := &server{
		config: streaming.DefaultConfig,
		provider: native.NewProvider(ctx,
			native.WithRecorder(recorderProvider),
			native.WithRecFilePathFactroy(func(id string) string { return id }),
		),
	}
	srv.installWs(http)

	return nil
}

func (p *server) installWs(http rest.GoRestfulContainer) {
	rest.WsRouteBuild(&rest.WsOption{
		Path:               "/remotecommand",
		GoRestfulContainer: http,
		Consumes:           []string{rest.MIME_ALL},
		Routes: []rest.WsRoute{
			{Method: "POST", SubPath: "/exec", Handle: p.exec},
			{Method: "POST", SubPath: "/attach", Handle: p.attach},
			{Method: "POST", SubPath: "/portforward", Handle: p.portForward},
		},
	})
}

func (p *server) exec(w http.ResponseWriter, req *http.Request, in *api.ExecRequest) error {
	klog.Info("entering exec")
	defer klog.Info("leaving exec")
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
