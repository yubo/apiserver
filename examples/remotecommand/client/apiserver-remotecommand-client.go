package main

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"time"

	dockerterm "github.com/moby/term"
	"github.com/yubo/apiserver/pkg/cmdcli"
	"github.com/yubo/apiserver/pkg/rest"
	"github.com/yubo/apiserver/pkg/streaming/api"
	"github.com/yubo/apiserver/tools/remotecommand"
	"github.com/yubo/golib/scheme"
	"github.com/yubo/golib/util/interrupt"
	"github.com/yubo/golib/util/term"
	"k8s.io/klog/v2"
)

type StreamOptions struct {
	Namespace     string
	PodName       string
	ContainerName string
	Stdin         bool
	TTY           bool
	// minimize unnecessary output
	Quiet bool
	// InterruptParent, if set, is used to handle interrupts while attached
	InterruptParent *interrupt.Handler

	cmdcli.IOStreams

	// for testing
	overrideStreams func() (io.ReadCloser, io.Writer, io.Writer)
	isTerminalIn    func(t term.TTY) bool
}

// ExecOptions declare the arguments accepted by the Exec command
type ExecOptions struct {
	StreamOptions
	//resource.FilenameOptions

	ResourceName     string
	Command          []string
	EnforceNamespace bool

	ParentCommandName       string
	EnableSuggestedCmdUsage bool

	//Builder         func() *resource.Builder
	//ExecutablePodFn polymorphichelpers.AttachablePodForObjectFunc
	//restClientGetter cmdcli.RESTClientGetter

	//Pod      *corev1.Pod
	Executor RemoteExecutor
	//PodClient     coreclient.PodsGetter
	GetPodTimeout time.Duration
	Config        *rest.Config
}

func (o *StreamOptions) SetupTTY() term.TTY {
	t := term.TTY{
		Parent: o.InterruptParent,
		Out:    o.Out,
	}

	if !o.Stdin {
		// need to nil out o.In to make sure we don't create a stream for stdin
		o.In = nil
		o.TTY = false
		return t
	}

	t.In = o.In
	if !o.TTY {
		return t
	}

	if o.isTerminalIn == nil {
		o.isTerminalIn = func(tty term.TTY) bool {
			return tty.IsTerminalIn()
		}
	}
	if !o.isTerminalIn(t) {
		o.TTY = false

		if o.ErrOut != nil {
			fmt.Fprintln(o.ErrOut, "Unable to use a TTY - input is not a terminal or the right kind of file")
		}

		return t
	}

	// if we get to here, the user wants to attach stdin, wants a TTY, and o.In is a terminal, so we
	// can safely set t.Raw to true
	t.Raw = true

	if o.overrideStreams == nil {
		// use dockerterm.StdStreams() to get the right I/O handles on Windows
		o.overrideStreams = dockerterm.StdStreams
	}
	stdin, stdout, _ := o.overrideStreams()
	o.In = stdin
	t.In = stdin
	if o.Out != nil {
		o.Out = stdout
		t.Out = stdout
	}

	return t
}

// RemoteExecutor defines the interface accepted by the Exec command - provided for test stubbing
type RemoteExecutor interface {
	Execute(method string, url *url.URL, config *rest.Config, stdin io.Reader, stdout, stderr io.Writer, tty bool, terminalSizeQueue remotecommand.TerminalSizeQueue) error
}

// DefaultRemoteExecutor is the standard implementation of remote command execution
type DefaultRemoteExecutor struct{}

func (*DefaultRemoteExecutor) Execute(method string, url *url.URL, config *rest.Config, stdin io.Reader, stdout, stderr io.Writer, tty bool, terminalSizeQueue remotecommand.TerminalSizeQueue) error {
	exec, err := remotecommand.NewSPDYExecutor(config, method, url)
	if err != nil {
		return err
	}
	return exec.Stream(remotecommand.StreamOptions{
		Stdin:             stdin,
		Stdout:            stdout,
		Stderr:            stderr,
		Tty:               tty,
		TerminalSizeQueue: terminalSizeQueue,
	})
}

func run() error {
	p := &ExecOptions{
		StreamOptions: StreamOptions{
			IOStreams: cmdcli.IOStreams{
				In:     os.Stdin,
				Out:    os.Stdout,
				ErrOut: os.Stderr,
			},
			Stdin: true,
			TTY:   true,
		},
		Executor: &DefaultRemoteExecutor{},
		Config: &rest.Config{
			Host:          "127.0.0.1:8080",
			ContentConfig: rest.ContentConfig{NegotiatedSerializer: scheme.Codecs},
		},
		Command: []string{"sh"},
	}
	containerName := "c08e76de8395"

	// ensure we can recover the terminal while attached
	t := p.SetupTTY()

	var sizeQueue remotecommand.TerminalSizeQueue
	if t.Raw {
		// this call spawns a goroutine to monitor/update the terminal size
		sizeQueue = t.MonitorSize(t.GetSize())

		// unset p.Err if it was previously set because both stdout and stderr go over p.Out when tty is
		// true
		p.ErrOut = nil
	}

	fn := func() error {
		restClient, err := rest.RESTClientFor(p.Config)
		if err != nil {
			return err
		}

		// TODO: consider abstracting into a client invocation or client helper
		req := restClient.Post().
			SubResource("exec")
		req.VersionedParams(&api.ExecRequest{
			ContainerId: containerName,
			Cmd:         p.Command,
			Stdin:       p.Stdin,
			Stdout:      p.Out != nil,
			Stderr:      p.ErrOut != nil,
			Tty:         t.Raw,
		}, scheme.ParameterCodec)

		return p.Executor.Execute(
			"POST",
			req.URL(),
			p.Config,
			p.In,
			p.Out,
			p.ErrOut,
			t.Raw,
			sizeQueue)
	}

	if err := t.Safe(fn); err != nil {
		return err
	}

	return nil

}

func main() {
	if err := run(); err != nil {
		klog.Error(err)
		os.Exit(1)
	}
}
