package native

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"

	dockerstdcopy "github.com/docker/docker/pkg/stdcopy"
	"github.com/yubo/golib/util/runtime"
	"github.com/yubo/golib/util/term"
	"k8s.io/klog/v2"
)

// operationTimeout is the error returned when the docker operations are timeout.
type operationTimeout struct {
	err error
}

func (e operationTimeout) Error() string {
	return fmt.Sprintf("operation timeout: %v", e.err)
}

// HandleResizing spawns a goroutine that processes the resize channel, calling resizeFunc for each
// remotecommand.TerminalSize received from the channel. The resize channel must be closed elsewhere to stop the
// goroutine.
func HandleResizing(resize <-chan term.TerminalSize, resizeFunc func(size term.TerminalSize)) {
	if resize == nil {
		return
	}

	go func() {
		defer runtime.HandleCrash()

		for size := range resize {
			if size.Height < 1 || size.Width < 1 {
				continue
			}
			resizeFunc(size)
		}
	}()
}

// contextError checks the context, and returns error if the context is timeout.
func contextError(ctx context.Context) error {
	if ctx.Err() == context.DeadlineExceeded {
		return operationTimeout{err: ctx.Err()}
	}
	return ctx.Err()
}

// redirectResponseToOutputStream redirect the response stream to stdout and stderr. When tty is true, all stream will
// only be redirected to stdout.
func redirectResponseToOutputStream(tty bool, outputStream, errorStream io.Writer, resp io.Reader) error {
	if outputStream == nil {
		outputStream = ioutil.Discard
	}
	if errorStream == nil {
		errorStream = ioutil.Discard
	}
	var err error
	if tty {
		_, err = io.Copy(outputStream, resp)
	} else {
		_, err = dockerstdcopy.StdCopy(outputStream, errorStream, resp)
	}
	return err
}

// holdConnection hold the HijackedResponse, redirect the inputStream to the connection, and redirect the response
// stream to stdout and stderr. NOTE: If needed, we could also add context in this function.
func holdConnection(tty bool, inputStream io.Reader, outputStream, errorStream io.Writer, resp *execResponse) error {
	klog.V(10).InfoS("entering holdConnection", "tty", tty, "inputStream", inputStream, "outputStream", outputStream, "errorStream", errorStream, "resp.stdout", resp.stdout, "resp.stdin", resp.stdin)
	receiveStdout := make(chan error)

	go func() {
		receiveStdout <- redirectResponseToOutputStream(tty, outputStream, errorStream, resp.stdout)
	}()

	stdinDone := make(chan struct{})
	go func() {
		if inputStream != nil {
			io.Copy(resp.stdin, inputStream)
		}
		close(stdinDone)
	}()

	select {
	case err := <-receiveStdout:
		klog.V(5).Infof("recevice err %s", err)
		return err
	case <-stdinDone:
		if outputStream != nil || errorStream != nil {
			return <-receiveStdout
		}
	}
	return nil
}

type exitError struct {
	Status *SessionStatus
}

func (d *exitError) String() string {
	return d.Error()
}

func (d *exitError) Error() string {
	return fmt.Sprintf("Error executing: %d", d.Status.ExitCode)
}

func (d *exitError) Exited() bool {
	return !d.Status.Running
}

func (d *exitError) ExitStatus() int {
	return d.Status.ExitCode
}
