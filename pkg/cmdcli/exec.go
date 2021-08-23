package cmdcli

import (
	"fmt"
	"io"
	"net/url"
	"os"

	mobyterm "github.com/moby/term"
	"github.com/yubo/apiserver/pkg/rest"
	"github.com/yubo/apiserver/pkg/streaming/api"
	"github.com/yubo/apiserver/tools/remotecommand"
	"github.com/yubo/golib/scheme"
	"github.com/yubo/golib/util/interrupt"
	"github.com/yubo/golib/util/term"
)

type ExecClient struct {
	verb string
	host string
	path string
}

func NewExecClient(verb, host, path string) *ExecClient {
	return &ExecClient{
		verb: verb,
		host: host,
		path: path,
	}
}

func (p *ExecClient) Exec(cmd string, args ...string) error {
	config := &rest.Config{
		Host:          p.host,
		ContentConfig: rest.ContentConfig{NegotiatedSerializer: scheme.Codecs},
	}

	client, err := rest.RESTClientFor(config)
	if err != nil {
		return err
	}

	o := &StreamOptions{
		IOStreams: IOStreams{
			In:     os.Stdin,
			Out:    os.Stdout,
			ErrOut: os.Stderr,
		},
		Stdin: true,
		TTY:   true,
	}
	// ensure we can recover the terminal while attached
	t := o.SetupTTY()

	var sizeQueue remotecommand.TerminalSizeQueue
	if t.Raw {
		// this call spawns a goroutine to monitor/update the terminal size
		sizeQueue = t.MonitorSize(t.GetSize())

		// unset p.Err if it was previously set because both stdout and stderr go over p.Out when tty is
		// true
		o.ErrOut = nil
	}

	req := client.
		Verb(p.verb).
		Prefix(p.path).
		VersionedParams(&api.ExecRequest{
			Cmd:    append([]string{cmd}, args...),
			Stdin:  o.Stdin,
			Stdout: o.Out != nil,
			Stderr: o.ErrOut != nil,
			Tty:    t.Raw,
		}, scheme.ParameterCodec)

	return t.Safe(func() error {
		return RemoteExecute(
			p.verb,
			req.URL(),
			config,
			o.In,
			o.Out,
			o.ErrOut,
			t.Raw,
			sizeQueue,
		)
	})
}

type StreamOptions struct {
	IOStreams

	Stdin bool
	TTY   bool

	// InterruptParent, if set, is used to handle interrupts while attached
	InterruptParent *interrupt.Handler

	// for testing
	overrideStreams func() (io.ReadCloser, io.Writer, io.Writer)
	isTerminalIn    func(t *term.TTY) bool
}

func (o *StreamOptions) SetupTTY() *term.TTY {
	t := &term.TTY{
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
		o.isTerminalIn = func(tty *term.TTY) bool {
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
		// use mobyterm.StdStreams() to get the right I/O handles on Windows
		o.overrideStreams = mobyterm.StdStreams
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

// DefaultRemoteExecutor is the standard implementation of remote command execution
func RemoteExecute(method string, url *url.URL, config *rest.Config, stdin io.Reader, stdout, stderr io.Writer, tty bool, terminalSizeQueue remotecommand.TerminalSizeQueue) error {
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
