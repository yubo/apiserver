package native

import (
	"context"
	"errors"
	"io"
	"os/exec"
	"strings"
	"sync"

	"github.com/yubo/golib/stream"
	"github.com/yubo/golib/term"
	"k8s.io/klog/v2"
)

const (
	defaultBufSize = 32 * 1024
)

type Session struct {
	sync.RWMutex
	*execConfig
	*Options

	ctx      context.Context
	cancel   context.CancelFunc
	id       string
	proxyTty *stream.ProxyTty
	running  bool
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
		ID:      p.id,
		Running: p.running,
	}
}

func (p *Session) Attach(in io.Reader, out, errOut io.WriteCloser, isTty bool, resize <-chan term.TerminalSize) error {
	// stream
	streamTty := stream.NewStreamTty(p.ctx, in, out, errOut, isTty, resize)

	if err := p.proxyTty.AddTty(streamTty); err != nil {
		return err
	}

	<-streamTty.Done()
	klog.V(6).Infof("attach done")

	return streamTty.Err()
}

func (p *Session) Close() error {
	p.running = false

	p.cancel()

	return nil
}

func (p *Session) init(ctx context.Context) error {
	if p.Timeout == 0 {
		p.ctx, p.cancel = context.WithCancel(ctx)
	} else {
		p.ctx, p.cancel = context.WithTimeout(ctx, p.Timeout)
	}

	p.proxyTty = stream.NewProxyTty(p.ctx, defaultBufSize)

	started := make(chan error)

	go func() {
		defer p.Close()

		if p.recorderProvider != nil {
			recorder, err := p.recorderProvider.Open(p.recFilePathFactory(p.id))
			if err != nil {
				started <- err
				return
			}
			defer recorder.Close()

			if err := recorder.Info([]byte(strings.Join(p.Cmd, " "))); err != nil {
				started <- err
				return
			}

			if err := p.proxyTty.AddRecorder(recorder); err != nil {
				started <- err
				return
			}
		}

		if len(p.Cmd) == 0 {
			started <- errors.New("empty command")
			return
		}

		pty, err := stream.NewCmdPty(exec.CommandContext(p.ctx, p.Cmd[0], p.Cmd[1:]...))
		if err != nil {
			started <- err
			return
		}
		defer pty.Close()

		p.running = true

		started <- nil

		err = <-p.proxyTty.CopyToPty(pty)
		klog.V(6).Infof("session %s exit %v", p.id, err)
	}()

	return <-started
}
