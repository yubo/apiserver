package native

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/yubo/apiserver/pkg/streaming"
	"github.com/yubo/golib/api/errors"
	"github.com/yubo/golib/stream"
	"github.com/yubo/golib/util/rand"
	"github.com/yubo/golib/util/term"
	"k8s.io/klog/v2"
)

const (
	defIdLent     = 10
	defGcInterval = time.Minute
)

func NewProvider(ctx context.Context, opts ...Opt) streaming.Provider {
	runtime, err := NewRuntime(ctx, opts...)
	if err != nil {
		panic(err)
	}

	return streaming.NewProvider(runtime)
}

func NewRuntime(ctx context.Context, opts ...Opt) (streaming.Runtime, error) {
	options := &Options{
		idLen:              defIdLent,
		gcInterval:         defGcInterval,
		recFilePathFactory: defRecFilePathFactory,
	}
	for _, opt := range opts {
		opt(options)
	}

	r := &streamingRuntime{
		ctx:      ctx,
		sessions: make(map[string]*Session),
		options:  options,
	}

	if err := r.start(); err != nil {
		return nil, err
	}

	return r, nil
}

type Options struct {
	recorderProvider   RecorderProvider
	gcInterval         time.Duration
	idLen              int
	recFilePathFactory func(sessionId string) string
}

type Opt func(*Options)

func WithRecorder(recorderProvider RecorderProvider) Opt {
	return func(o *Options) {
		o.recorderProvider = recorderProvider
	}
}

func WithIdLen(idLen int) Opt {
	return func(o *Options) {
		o.idLen = idLen
	}
}

func WithGcInterval(gcInterval time.Duration) Opt {
	return func(o *Options) {
		o.gcInterval = gcInterval
	}
}

func WithRecFilePathFactroy(factory func(sessionId string) string) Opt {
	return func(o *Options) {
		o.recFilePathFactory = factory
	}
}

var _ streaming.Runtime = &streamingRuntime{}

type streamingRuntime struct {
	sync.RWMutex
	sessions map[string]*Session
	ctx      context.Context
	options  *Options
}

type execConfig struct {
	User       string   // User that will run the command
	Detach     bool     // Execute in detach mode
	DetachKeys string   // Escape keys for detach
	Env        []string // Environment variables
	WorkingDir string   // Working directory
	Cmd        []string // Execution commands and args
	Timeout    time.Duration
	recorder   stream.Recorder
}

func (p *streamingRuntime) Exec(containerID string, cmd []string, in io.Reader, out, errOut io.WriteCloser, isTty bool, resize <-chan term.TerminalSize) error {

	session, err := p.newSession(&execConfig{Cmd: cmd, Timeout: 0})
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
		ticker := time.NewTicker(p.options.gcInterval)
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
		code := rand.String(p.options.idLen)
		if _, exists := p.sessions[code]; !exists {
			return code, nil
		}
	}
	return "", fmt.Errorf("failed to generate unique id")
}

// TODO
func (p *streamingRuntime) newSession(config *execConfig) (*Session, error) {
	p.Lock()
	defer p.Unlock()

	id, err := p.uniqueID()
	if err != nil {
		return nil, err
	}

	s := &Session{
		execConfig: config,
		id:         id,
		Options:    p.options,
	}

	if err := s.init(p.ctx); err != nil {
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

func defRecFilePathFactory(sessionId string) string {
	// should set nodeName by uuid.SetNodeInterface(name string)
	file, err := uuid.NewUUID()
	if err != nil {
		panic(err)
	}
	return file.String()
}
