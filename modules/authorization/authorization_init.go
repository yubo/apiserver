package authorization

import (
	"context"

	"github.com/yubo/apiserver/pkg/authorization/authorizer"
	"github.com/yubo/apiserver/pkg/options"
	"github.com/yubo/golib/proc"
	pconfig "github.com/yubo/golib/proc/config"
	"k8s.io/klog/v2"
)

const (
	moduleName = "authorization"
)

var (
	_server = &authorization{name: moduleName}
	hookOps = []proc.HookOps{{
		Hook:        _server.init,
		Owner:       moduleName,
		HookNum:     proc.ACTION_TEST,
		Priority:    proc.PRI_SYS_INIT,
		SubPriority: options.PRI_M_HTTP,
	}, {
		Hook:        _server.init,
		Owner:       moduleName,
		HookNum:     proc.ACTION_START,
		Priority:    proc.PRI_SYS_INIT,
		SubPriority: options.PRI_M_AUTHZ,
	}, {
		Hook:        _server.start,
		Owner:       moduleName,
		HookNum:     proc.ACTION_START,
		Priority:    proc.PRI_SYS_START,
		SubPriority: options.PRI_M_AUTHZ,
	}, {
		Hook:        _server.stop,
		Owner:       moduleName,
		HookNum:     proc.ACTION_STOP,
		Priority:    proc.PRI_SYS_START,
		SubPriority: options.PRI_M_AUTHZ,
	}}
	_config *config
	Config  *config
)

type authorization struct {
	name          string
	config        *config
	authorization *Authorization

	ctx       context.Context
	cancel    context.CancelFunc
	stoppedCh chan struct{}
}

func (p *authorization) Authorizer() authorizer.Authorizer {
	return p.authorization.Authorizer
}

func (p *authorization) RuleResolver() authorizer.RuleResolver {
	return p.authorization.RuleResolver
}

func (p *authorization) init(ops *proc.HookOps) (err error) {
	ctx, configer := ops.ContextAndConfiger()
	p.ctx, p.cancel = context.WithCancel(ctx)

	cf := newConfig()
	klog.Infof(">>> %s", cf)
	klog.Infof(">>> %s", _config)
	klog.Infof(">>> %s", _config.Changed())
	if err := configer.ReadYaml(p.name, cf,
		pconfig.WithOverride(_config.Changed())); err != nil {
		return err
	}
	p.config = cf

	if p.authorization, err = newAuthorization(ctx, p.config); err != nil {
		return err
	}

	ops.SetContext(options.WithAuthz(ctx, p))

	return nil
}

func (p *authorization) start(ops *proc.HookOps) error {
	//if err := p.server.Start(p.ctx.Done(), p.stoppedCh); err != nil {
	//	return err
	//}

	return nil
}

func (p *authorization) stop(ops *proc.HookOps) error {
	if p.cancel == nil {
		return nil
	}

	p.cancel()

	//<-p.stoppedCh

	return nil
}

func init() {
	proc.RegisterHooks(hookOps)

	_config = newConfig()
	Config = _config
	_config.addFlags(proc.NamedFlagSets().FlagSet("authorization"))
}
