package native

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/yubo/apiserver/pkg/streaming"
	"github.com/yubo/golib/api/errors"
	"github.com/yubo/golib/util/rand"
	"github.com/yubo/golib/util/term"
	"k8s.io/klog/v2"
)

const (
	defaultBufSize = 32 * 1024
)

var (
	idLen      = 10
	gcInterval = time.Minute
)

func NewProvider(ctx context.Context, recFd io.WriteCloser) streaming.Provider {
	if runtime, err := NewRuntime(ctx, recFd); err != nil {
		panic(err)
	} else {
		return streaming.NewProvider(runtime)
	}
}

func NewRuntime(ctx context.Context, recFd io.WriteCloser) (streaming.Runtime, error) {
	r := &streamingRuntime{
		ctx:      ctx,
		sessions: make(map[string]*Session),
		recFd:    recFd,
	}

	if err := r.start(); err != nil {
		return nil, err
	}

	return r, nil
}

type streamingRuntime struct {
	sync.RWMutex
	sessions map[string]*Session
	ctx      context.Context
	recFd    io.WriteCloser
}

var _ streaming.Runtime = &streamingRuntime{}

func (p *streamingRuntime) Exec(containerID string, cmd []string, in io.Reader, out, errOut io.WriteCloser, isTty bool, resize <-chan term.TerminalSize) error {

	session, err := p.newSession(&ExecConfig{Cmd: cmd, Timeout: 0})
	if err != nil {
		return fmt.Errorf("failed to exec - Exec setup failed - %v", err)
	}

	return session.Attach(in, out, errOut, isTty, resize)
}

func (p *streamingRuntime) Attach(sessionID string, in io.Reader, out, errOut io.WriteCloser, isTty bool, resize <-chan term.TerminalSize) error {
	session, err := p.checkSessionStatus(sessionID)
	if err != nil {
		return err
	}
	return session.Attach(in, out, errOut, isTty, resize)
}

func (r *streamingRuntime) PortForward(podSandboxID string, port int32, stream io.ReadWriteCloser) error {
	return fmt.Errorf("unsupported port forward")
}

func (p *streamingRuntime) start() error {
	go func() {
		ticker := time.NewTicker(gcInterval)
		defer ticker.Stop()

		for {
			select {
			case <-p.ctx.Done():
				return
			case <-ticker.C:
				// TODO: add sessions gc
				klog.V(6).Infof("todo gc")
			}
		}
	}()

	return nil
}

func (p *streamingRuntime) getSession(id string) (*Session, error) {
	p.RLock()
	defer p.RUnlock()

	s, ok := p.sessions[id]
	if !ok {
		return nil, errors.NewNotFound("session id: " + id)
	}
	return s, nil
}

// unsafe
func (p *streamingRuntime) uniqueID() (string, error) {
	const maxTries = 10
	// Number of bytes to be tokenLen when base64 encoded.
	for i := 0; i < maxTries; i++ {
		code := rand.String(idLen)
		if _, exists := p.sessions[code]; !exists {
			return code, nil
		}
	}
	return "", fmt.Errorf("failed to generate unique id")
}

// TODO
func (p *streamingRuntime) newSession(config *ExecConfig) (*Session, error) {
	p.Lock()
	defer p.Unlock()

	id, err := p.uniqueID()
	if err != nil {
		return nil, err
	}

	s := &Session{
		ExecConfig: config,
		id:         id,
	}

	if err := s.init(p.ctx, p.recFd); err != nil {
		return nil, err
	}

	p.sessions[id] = s

	klog.V(6).Infof("seesionid %s", id)
	return s, nil
}

func (p *streamingRuntime) checkSessionStatus(id string) (*Session, error) {
	session, err := p.getSession(id)
	if err != nil {
		return nil, err
	}
	if !session.Status().Running {
		return nil, fmt.Errorf("session not running (%s)", id)
	}
	return session, nil

}
