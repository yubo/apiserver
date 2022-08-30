package session

import (
	"context"

	"github.com/yubo/apiserver/pkg/models"
	"github.com/yubo/apiserver/pkg/options"
	"github.com/yubo/golib/proc"
)

// depend "github.com/yubo/apiserver/pkg/models/register"

const (
	moduleName = "session"
)

var (
	module  = &sessionModule{}
	hookOps = []proc.HookOps{{
		Hook:        module.init,
		Owner:       moduleName,
		HookNum:     proc.ACTION_START,
		Priority:    proc.PRI_SYS_INIT,
		SubPriority: options.PRI_M_AUTHN - 1,
	}}
)

type sessionModule struct{}

func (p *sessionModule) init(ctx context.Context) error {
	cf := NewConfig()
	if err := proc.ReadConfig(moduleName, cf); err != nil {
		return err
	}

	manager, err := NewSessionManager(cf, WithCtx(ctx), WithModel(NewSessionConn()))
	if err != nil {
		return err
	}

	manager.GC()

	options.WithSessionManager(ctx, manager)

	return nil
}
func Register() {
	proc.RegisterHooks(hookOps)
	proc.AddConfig(moduleName, NewConfig(), proc.WithConfigGroup("session"))
}

func init() {
	models.Register(&SessionConn{})
}