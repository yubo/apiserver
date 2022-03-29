package reporter

import (
	"context"

	"github.com/yubo/golib/proc"
	"k8s.io/klog/v2"
)

const (
	moduleName = "version.reporter"
)

var (
	_module         = &module{}
	reporterHookOps = []proc.HookOps{{
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

type module struct {
	ctx    context.Context
	cancel context.CancelFunc
}

func (p *module) start(ctx context.Context) error {
	if p.cancel != nil {
		p.cancel()
	}
	p.ctx, p.cancel = context.WithCancel(ctx)

	reporter := &buildReporter{}

	if err := reporter.Start(); err != nil {
		return err
	}

	go func() {
		<-p.ctx.Done()
		reporter.Stop()
	}()

	return nil
}

func (p *module) stop(ctx context.Context) error {
	klog.Info("stop")
	p.cancel()
	return nil
}

func Register() {
	proc.RegisterHooks(reporterHookOps)
}
