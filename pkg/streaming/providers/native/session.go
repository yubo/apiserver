package native

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/containerd/console"
	"github.com/yubo/golib/util/term"
	"k8s.io/klog/v2"
)

type Session struct {
	sync.RWMutex

	ctx      context.Context
	cancel   context.CancelFunc
	resp     *execResponse
	config   *execConfig
	id       string
	running  bool
	exitCode int
	pid      int
}

type SessionStatus struct {
	ID       string `json:"id"`
	Running  bool   `json:"running"`
	ExitCode int    `json:"exitCode"`
	Pid      int    `json:"pid"`
}

func (p *Session) Status() *SessionStatus {
	p.RLock()
	defer p.RUnlock()

	return &SessionStatus{
		ID:       p.id,
		Running:  p.running,
		ExitCode: p.exitCode,
		Pid:      p.pid,
	}
}

// TODO: check min size
func (p *Session) resizeTTY(height, width uint16) error {
	if p.resp != nil && p.resp.console != nil {
		return p.resp.console.Resize(console.WinSize{
			Height: height,
			Width:  width,
		})
	}
	return nil
}

func NewSession(config *execConfig) (*Session, error) {
	return &Session{
		config: config,
	}, nil
}

func (p *Session) Attach(stdin io.Reader, stdout, stderr io.WriteCloser, tty bool, resize <-chan term.TerminalSize) error {
	// Have to start this before the call to client.AttachToSession because client.AttachToSession is a blocking
	// call :-( Otherwise, resize events don't get processed and the terminal never resizes.
	HandleResizing(resize, func(size term.TerminalSize) {
		p.resizeTTY(size.Height, size.Width)
	})

	// TODO(random-liu): Do we really use the *Logs* field here?
	opts := &AttachOptions{
		Stream: true,
		Stdin:  stdin != nil,
		Stdout: stdout != nil,
		Stderr: stderr != nil,
	}
	sopts := &StreamOptions{
		InputStream:  stdin,
		OutputStream: stdout,
		ErrorStream:  stderr,
		RawTerminal:  tty,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	resp, err := p.attach(ctx, opts)
	if ctxErr := contextError(ctx); ctxErr != nil {
		return ctxErr
	}
	if err != nil {
		return err
	}
	defer resp.Close()

	return holdConnection(sopts.RawTerminal, sopts.InputStream, sopts.OutputStream, sopts.ErrorStream, resp)

}

// Exec executes the cmd in container using the Docker's Exec API
func (p *Session) Exec(ctx context.Context, cmd []string, stdin io.Reader, stdout, stderr io.WriteCloser, tty bool, resize <-chan term.TerminalSize, timeout time.Duration) error {
	done := make(chan struct{})
	defer close(done)

	execStarted := make(chan struct{})
	go func() {
		select {
		case <-execStarted:
			// client.StartExec has started the exec, so we can start resizing
		case <-done:
			// ExecInContainer has returned, so short-circuit
			return
		}

		HandleResizing(resize, func(size term.TerminalSize) {
			p.resizeTTY(size.Height, size.Width)
		})
	}()

	startOpts := &ExecStartCheck{Tty: tty}
	streamOpts := &StreamOptions{
		InputStream:  stdin,
		OutputStream: stdout,
		ErrorStream:  stderr,
		RawTerminal:  tty,
		ExecStarted:  execStarted,
	}

	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	// startExec is a blocking call, so we need to run it concurrently and catch
	// its error in a channel
	execErr := make(chan error, 1)
	go func() {
		execErr <- p.startExec(startOpts, streamOpts)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-execErr:
		if err != nil {
			return err
		}
	}

	// InspectExec may not always return latest state of exec, so call it a few times until
	// it returns an exec inspect that shows that the process is no longer running.
	retries := 0
	maxRetries := 5
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		status := p.Status()
		if !status.Running {
			if status.ExitCode != 0 {
				return &exitError{Status: status}
			}

			return nil
		}

		retries++
		if retries == maxRetries {
			klog.Errorf("Exec session %s terminated but process still running!", p.id)
			return nil
		}

		<-ticker.C
	}

}

func (p *Session) Detach() {
}

func (p *Session) startExec(opts *ExecStartCheck, sopts *StreamOptions) error {
	resp, err := func() (*execResponse, error) {
		p.Lock()
		defer p.Unlock()

		p.ctx, p.cancel = context.WithCancel(context.Background())

		resp, err := p.exec()
		if ctxErr := contextError(p.ctx); ctxErr != nil {
			return nil, ctxErr
		}
		if err != nil {
			return nil, err
		}
		p.resp = resp
		//defer resp.Close()

		if sopts.ExecStarted != nil {
			// Send a message to the channel indicating that the exec has started. This is needed so
			// interactive execs can handle resizing correctly - the request to resize the TTY has to happen
			// after the call to d.client.ExecAttach, and because d.holdHijackedConnection below
			// blocks, we use sopts.ExecStarted to signal the caller that it's ok to resize.
			sopts.ExecStarted <- struct{}{}
		}

		return resp, nil

	}()
	if err != nil {
		return err
	}

	//TODO: broadcast
	holdConnection(sopts.RawTerminal || opts.Tty, sopts.InputStream, sopts.OutputStream, sopts.ErrorStream, resp)

	return nil
}

// TODO
func (p *Session) exec() (*execResponse, error) {
	cf := p.config

	if len(cf.Cmd) == 0 {
		return nil, errors.New("empty command")
	}

	cmd := exec.CommandContext(p.ctx, cf.Cmd[0], cf.Cmd[1:]...)
	cmd.Env = cf.Env

	console, slavePath, err := console.NewPty()
	if err != nil {
		return nil, err
	}

	slave, err := os.OpenFile(slavePath, os.O_RDWR, 0)
	if err != nil {
		console.Close()
		return nil, err
	}

	cmd.Stdin = slave
	cmd.Stdout = slave
	cmd.Stderr = slave

	if err := cmd.Start(); err != nil {
		slave.Close()
		console.Close()
		return nil, err
	}

	p.running = true
	p.pid = cmd.Process.Pid

	klog.V(10).InfoS("started", "cmd", cmd, "pid", p.pid)
	go func() {
		if err := p.wait(); err != nil {
			klog.Error("session %s wait err %s", p.pid, err)
		}

		p.Close()
	}()

	resp := &execResponse{
		stdin:   console,
		stdout:  console,
		stderr:  console,
		session: p,

		console: console,
		slave:   slave,
		cmd:     cmd,
	}

	p.resp = resp

	return resp, nil
}

// TODO
func (p *Session) attach(ctx context.Context, opts *AttachOptions) (*execResponse, error) {
	return nil, nil
}

func (p *Session) wait() error {
	klog.V(10).InfoS("entering wait()", "pid", p.pid)
	var exitCode int

	defer func() {
		p.Lock()
		p.running = false
		p.exitCode = exitCode
		p.Unlock()
	}()

	if p.resp.cmd == nil {
		return errors.New("exec: not started")
	}

	err := p.resp.cmd.Wait()
	if err == nil {
		return nil

	}
	if status, ok := err.(*exec.ExitError); ok {
		exitCode = status.ExitCode()
	}

	return err
}

func (p *Session) Close() {
	p.Lock()
	defer p.Unlock()

	if p.running {
		// kill cmd indrect
		proc := p.resp.cmd.Process
		if err := proc.Kill(); err != nil {
			klog.Error("kill %s err %s", proc.Pid, err)
		}
		p.running = false
		p.exitCode = -1
	}

	p.resp.Close()

	if p.cancel != nil {
		p.cancel()
	}

}
