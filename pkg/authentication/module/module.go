package module

import (
	"context"

	"github.com/yubo/apiserver/pkg/authentication/authenticator"
	"github.com/yubo/apiserver/pkg/options"
	"github.com/yubo/golib/proc"
	pconfig "github.com/yubo/golib/proc/config"
)

const (
	moduleName = "authentication"
)

var (
	_server = &authentication{name: moduleName}
	hookOps = []proc.HookOps{{
		Hook:        _server.init,
		Owner:       moduleName,
		HookNum:     proc.ACTION_TEST,
		Priority:    proc.PRI_SYS_INIT,
		SubPriority: options.PRI_M_AUTHN,
	}, {
		Hook:        _server.init,
		Owner:       moduleName,
		HookNum:     proc.ACTION_START,
		Priority:    proc.PRI_SYS_INIT,
		SubPriority: options.PRI_M_AUTHN,
	}, {
		Hook:        _server.stop,
		Owner:       moduleName,
		HookNum:     proc.ACTION_STOP,
		Priority:    proc.PRI_SYS_START,
		SubPriority: options.PRI_M_AUTHN,
	}}
	_config *config
)

type authentication struct {
	name           string
	config         *config
	authentication *Authentication

	ctx       context.Context
	cancel    context.CancelFunc
	stoppedCh chan struct{}
}

func (p *authentication) APIAudiences() authenticator.Audiences {
	return p.authentication.APIAudiences
}

func (p *authentication) Authenticator() authenticator.Request {
	return p.authentication.Authenticator
}

func (p *authentication) init(ops *proc.HookOps) (err error) {
	ctx, configer := ops.ContextAndConfiger()
	p.ctx, p.cancel = context.WithCancel(ctx)

	cf := defaultConfig()
	if err := configer.ReadYaml(p.name, cf,
		pconfig.WithOverride(_config.changed())); err != nil {
		return err
	}
	p.config = cf

	if p.authentication, err = newAuthentication(p.ctx, p.config); err != nil {
		return err
	}

	ops.SetContext(options.WithAuthn(ctx, p))
	return nil
}

func (p *authentication) stop(ops *proc.HookOps) error {
	if p.cancel == nil {
		return nil
	}

	p.cancel()

	//<-p.stoppedCh

	return nil
}

func Register() {
	proc.RegisterHooks(hookOps)

	_config = defaultConfig()
	_config.addFlags(proc.NamedFlagSets().FlagSet("authentication"))

}
