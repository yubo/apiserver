package apiserver

import (
	"context"

	"github.com/go-openapi/spec"
	"github.com/yubo/apiserver/pkg/options"
	"github.com/yubo/apiserver/pkg/rest"
	"github.com/yubo/golib/proc"
	"k8s.io/klog/v2"
)

const (
	moduleName = "apiserver"
	APIPath    = "/apidocs.json"
)

var (
	_module = &apiserver{name: moduleName}
	hookOps = []proc.HookOps{{
		Hook:        _module.init,
		Owner:       moduleName,
		HookNum:     proc.ACTION_START,
		Priority:    proc.PRI_SYS_INIT,
		SubPriority: options.PRI_M_HTTP,
	}, {
		Hook:        _module.start,
		Owner:       moduleName,
		HookNum:     proc.ACTION_START,
		Priority:    proc.PRI_SYS_START,
		SubPriority: options.PRI_M_HTTP,
	}, {
		Hook:        _module.stop,
		Owner:       moduleName,
		HookNum:     proc.ACTION_STOP,
		Priority:    proc.PRI_SYS_START,
		SubPriority: options.PRI_M_HTTP,
	}}
)

type apiserver struct {
	name   string
	config *config
	server *Server

	ctx       context.Context
	cancel    context.CancelFunc
	stoppedCh chan struct{}
}

func (p *apiserver) init(ops *proc.HookOps) (err error) {
	ctx, c := ops.ContextAndConfiger()
	p.ctx, p.cancel = context.WithCancel(ctx)

	//c1 := c.GetConfiger(moduleName)

	cf := newConfig()
	if err := c.Read(moduleName, cf); err != nil {
		return err
	}
	p.config = cf
	klog.V(10).Infof("%s config: %s\n", p.name, cf)

	if err := p.serverInit(); err != nil {
		return err
	}

	ops.SetContext(options.WithGenericServer(ctx, p))

	return nil
}

func (p *apiserver) start(ops *proc.HookOps) error {
	ctx, _ := ops.ContextAndConfiger()
	rest.InstallApiDocs(
		p.server.Handler.GoRestfulContainer,
		spec.InfoProps{Title: proc.NameFrom(ctx)},
		APIPath,
	)

	if err := p.server.Start(p.ctx.Done(), p.stoppedCh); err != nil {
		return err
	}

	return nil
}

func (p *apiserver) stop(ops *proc.HookOps) error {
	if p.cancel == nil {
		return nil
	}

	p.cancel()

	<-p.stoppedCh

	return nil
}

func Register() {
	proc.RegisterHooks(hookOps)

	proc.RegisterFlags(moduleName, "apiserver", newConfig())
}
