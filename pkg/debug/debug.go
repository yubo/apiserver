package debug

import (
	"context"

	"github.com/yubo/golib/proc"
)

const (
	moduleName = "debug"
)

type config struct {
	Address *string
	Pprof   bool
	Expvar  bool
	Metrics bool
}

type debugModule struct {
	config    *config
	name      string
	stoppedCh <-chan struct{}
	ctx       context.Context
	cancel    context.CancelFunc
}

var (
	_module = &debugModule{name: moduleName}
	hookOps = []proc.HookOps{{
		Hook:     _module.start,
		Owner:    moduleName,
		HookNum:  proc.ACTION_START,
		Priority: proc.PRI_MODULE,
	}, {
		Hook:     _module.stop,
		Owner:    moduleName,
		HookNum:  proc.ACTION_STOP,
		Priority: proc.PRI_MODULE,
	}}
)

func (p *debugModule) start(ctx context.Context) error {
	c := proc.ConfigerMustFrom(ctx)

	cf := &config{}
	if err := c.Read(p.name, cf); err != nil {
		return err
	}
	p.config = cf

	if cf.Address == nil {
		return nil
	}

	p.ctx, p.cancel = context.WithCancel(ctx)

	server, err := newServer(cf)
	if err != nil {
		return err
	}

	stoppedCh, err := server.start(p.ctx)
	if err != nil {
		return err
	}

	p.stoppedCh = stoppedCh

	return nil
}

func (p *debugModule) stop(ctx context.Context) error {
	if p.cancel == nil {
		return nil
	}

	p.cancel()
	<-p.stoppedCh

	return nil
}

func Register() {
	proc.RegisterHooks(hookOps)
}
