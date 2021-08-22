package native

import (
	"context"
	"fmt"
	"io"

	"github.com/yubo/apiserver/pkg/streaming"
	"github.com/yubo/golib/util/term"
)

type streamingRuntime struct {
	controller *Controller
}

var _ streaming.Runtime = &streamingRuntime{}

func (p *streamingRuntime) Exec(containerID string, cmd []string, in io.Reader, out, err io.WriteCloser, tty bool, resize <-chan term.TerminalSize) error {
	tty = true
	session, e := p.controller.newSession(&execConfig{
		Cmd:          cmd,
		AttachStdin:  in != nil,
		AttachStdout: out != nil,
		AttachStderr: err != nil,
		Tty:          tty,
	})
	if e != nil {
		return fmt.Errorf("failed to exec - Exec setup failed - %v", e)
	}

	return session.Exec(context.TODO(), cmd, in, out, err, tty, resize, 0)
}

func (p *streamingRuntime) Attach(execID string, in io.Reader, out, errw io.WriteCloser, tty bool, resize <-chan term.TerminalSize) error {
	session, err := p.controller.checkSessionStatus(execID)
	if err != nil {
		return err
	}
	return session.Attach(in, out, errw, tty, resize)
}

func (r *streamingRuntime) PortForward(podSandboxID string, port int32, stream io.ReadWriteCloser) error {
	return fmt.Errorf("unsupported port forward")
}
