package session

import (
	"context"

	"github.com/yubo/apiserver/pkg/models"
	"github.com/yubo/apiserver/pkg/proc"
	v1 "github.com/yubo/apiserver/pkg/proc/api/v1"
	"github.com/yubo/apiserver/pkg/session/filter"
)

// depend "github.com/yubo/apiserver/pkg/models/register"

const (
	moduleName = "session"
)

var (
	module  = &sessionModule{}
	hookOps = []v1.HookOps{{
		Hook:        module.init,
		Owner:       moduleName,
		HookNum:     v1.ACTION_START,
		Priority:    v1.PRI_SYS_INIT,
		SubPriority: v1.PRI_M_AUTHN - 1,
	}}
)

type sessionModule struct{}

func (p *sessionModule) init(ctx context.Context) error {
	cf := newConfig()
	if err := proc.ReadConfig(moduleName, cf); err != nil {
		return err
	}

	manager := newManager(ctx, cf, nil)

	manager.GC()

	filter.SetManager(manager)

	return nil
}

func Register() {
	proc.RegisterHooks(hookOps)
	proc.AddConfig(moduleName, newConfig(), proc.WithConfigGroup("session"))
	models.Register(&SessionConn{})
}
