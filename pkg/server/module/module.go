package server

import (
	"context"

	"github.com/yubo/apiserver/pkg/options"
	"github.com/yubo/apiserver/pkg/server"
	"github.com/yubo/apiserver/pkg/server/config"
	"github.com/yubo/golib/configer"
	"github.com/yubo/golib/proc"
	"k8s.io/klog/v2"
)

const (
	moduleName = "apiserver"
	APIPath    = "/apidocs.json"
)

var (
	_module = &module{name: moduleName}
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

var _ server.APIServer = &module{}

type module struct {
	name   string
	config *server.Config

	ctx       context.Context
	cancel    context.CancelFunc
	stoppedCh chan struct{}
}

func (p *module) init(ctx context.Context) (err error) {
	c := proc.ConfigerMustFrom(ctx)

	p.ctx, p.cancel = context.WithCancel(ctx)

	cf := config.NewConfig()
	if err := c.Read(moduleName, cf); err != nil {
		return err
	}

	p.config = cf.NewServerConfig()

	if err := p.serverInit(cf); err != nil {
		return err
	}

	options.WithAPIServer(ctx, p)

	return nil
}

func (p *module) Address() string {
	return p.config.SecureServing.Listener.Addr().String()
}

func (p *module) start(ctx context.Context) error {
	if err := p.Start(p.ctx.Done(), p.stoppedCh); err != nil {
		return err
	}

	p.Info()

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

func (p *module) Info() {
	if !klog.V(10).Enabled() {
		return
	}
	for _, path := range p.config.Handler.ListedPaths() {
		klog.Infof("apiserver path %s", path)
	}
}

func RegisterHooks() {
	proc.RegisterHooks(hookOps)
}

func RegisterFlags() {
	cf := config.NewConfig()
	proc.RegisterFlags(moduleName, "APIServer", cf, configer.WithTags(cf.Tags()))
}

func Register() {
	RegisterHooks()
	RegisterFlags()
}
