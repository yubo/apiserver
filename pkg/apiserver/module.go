package apiserver

import (
	"context"
	"fmt"

	"github.com/go-openapi/spec"
	"github.com/yubo/apiserver/pkg/options"
	apirequest "github.com/yubo/apiserver/pkg/request"
	"github.com/yubo/apiserver/pkg/rest"
	"github.com/yubo/golib/proc"
	utilwaitgroup "github.com/yubo/golib/util/waitgroup"
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

	// handlerChainWaitGroup allows you to wait for all chain handlers exit after the server shutdown.
	handlerChainWaitGroup *utilwaitgroup.SafeWaitGroup

	// Predicate which is true for paths of long-running http requests
	longRunningFunc apirequest.LongRunningRequestCheck

	// handler holds the handlers being used by this API server
	handler *APIServerHandler

	// apiServerID is the ID of this API server
	apiServerID string

	// requestInfoResolver is used to assign attributes (used by admission and authorization) based on a request URL.
	// Use-cases that are like kubelets may need to customize this.
	requestInfoResolver apirequest.RequestInfoResolver

	ctx       context.Context
	cancel    context.CancelFunc
	stoppedCh chan struct{}
}

func (p *apiserver) init(ctx context.Context) (err error) {
	c := proc.ConfigerMustFrom(ctx)

	p.ctx, p.cancel = context.WithCancel(ctx)

	cf := newConfig()
	if err := c.Read(moduleName, cf); err != nil {
		return err
	}
	p.config = cf

	if err := p.serverInit(); err != nil {
		return err
	}

	options.WithApiServer(ctx, p)

	return nil
}

func (p *apiserver) Address() string {
	return fmt.Sprintf("%s:%d", p.config.Host, p.config.Port)
}

func (p *apiserver) start(ctx context.Context) error {
	rest.InstallApiDocs(
		p.handler.GoRestfulContainer,
		spec.InfoProps{Title: proc.NameFrom(ctx)},
		APIPath,
	)

	if err := p.Start(p.ctx.Done(), p.stoppedCh); err != nil {
		return err
	}

	p.Info()

	return nil
}

func (p *apiserver) stop(ctx context.Context) error {
	if p.cancel == nil {
		return nil
	}

	p.cancel()

	<-p.stoppedCh

	return nil
}

func (p *apiserver) Info() {
	if !klog.V(10).Enabled() {
		return
	}
	for _, path := range p.handler.ListedPaths() {
		klog.Infof("apiserver path %s", path)
	}
}

func RegisterHooks() {
	proc.RegisterHooks(hookOps)
}

func RegisterFlags() {
	proc.RegisterFlags(moduleName, "apiserver", newConfig())
}

func Register() {
	RegisterHooks()
	RegisterFlags()
}
