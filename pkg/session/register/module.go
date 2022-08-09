package register

// depend "github.com/yubo/apiserver/pkg/models/register"
import (
	"context"

	"github.com/yubo/apiserver/pkg/options"
	"github.com/yubo/apiserver/pkg/session"
	"github.com/yubo/apiserver/pkg/session/types"
	"github.com/yubo/golib/proc"
)

const (
	moduleName = "session"
)

type module struct {
	config  *session.Config
	name    string
	manager types.SessionManager
}

var (
	_module = &module{name: moduleName}
	hookOps = []proc.HookOps{{
		Hook:        _module.init,
		Owner:       moduleName,
		HookNum:     proc.ACTION_START,
		Priority:    proc.PRI_SYS_INIT,
		SubPriority: options.PRI_M_AUTHN - 1,
	}, {
		Hook:        _module.postStart,
		Owner:       moduleName,
		HookNum:     proc.ACTION_START,
		Priority:    proc.PRI_SYS_POSTSTART,
		SubPriority: options.PRI_M_AUTHN - 1,
	}}
)

func (p *module) init(ctx context.Context) error {
	cf := newConfig()
	if err := proc.ReadConfig(p.name, cf); err != nil {
		return err
	}
	p.config = cf

	var err error
	if p.manager, err = startSession(cf, ctx); err != nil {
		return err
	}

	options.WithSessionManager(ctx, p.manager)

	return nil
}

func startSession(cf *session.Config, ctx context.Context) (types.SessionManager, error) {
	opts := []session.Option{
		session.WithCtx(ctx),
		session.WithModel(session.NewSession()),
	}
	return session.NewSessionManager(cf, opts...)
}

func (p *module) postStart(ctx context.Context) error {
	p.manager.GC()
	return nil
}

func newConfig() *session.Config {
	return session.NewConfig()
}

func Register() {
	proc.RegisterHooks(hookOps)
	proc.AddConfig(moduleName, newConfig(), proc.WithConfigGroup("session"))
}
