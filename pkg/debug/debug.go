package debug

import (
	"context"

	"github.com/yubo/golib/proc"
)

const (
	moduleName = "debug"
)

type config struct {
	Address     *string `json:"address" default:"loalhost:8080"`
	Pprof       bool    `json:"pprof" default:"true"`
	Expvar      bool    `json:"expvar" default:"true"`
	Metrics     bool    `json:"metrics" default:"true"`
	MetricsPath string  `json:"metricsPath" default:"/debug/metrics"`
}

type module struct {
	config    *config
	name      string
	stoppedCh <-chan struct{}
	ctx       context.Context
	cancel    context.CancelFunc
}

var (
	this    = &module{name: moduleName}
	hookOps = []proc.HookOps{{
		Hook:     this.start,
		Owner:    moduleName,
		HookNum:  proc.ACTION_START,
		Priority: proc.PRI_MODULE,
	}, {
		Hook:     this.stop,
		Owner:    moduleName,
		HookNum:  proc.ACTION_STOP,
		Priority: proc.PRI_MODULE,
	}}
)

func (p *module) start(ctx context.Context) error {
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

func (p *module) stop(ctx context.Context) error {
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
