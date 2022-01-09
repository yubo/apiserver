package session

import (
	"context"
	"fmt"

	"github.com/yubo/apiserver/pkg/options"
	"github.com/yubo/apiserver/pkg/session"
	"github.com/yubo/golib/configer"
	"github.com/yubo/golib/proc"
)

const (
	moduleName = "session"
)

type module struct {
	config *session.Config
	name   string
}

var (
	_module = &module{name: moduleName}
	hookOps = []proc.HookOps{{
		Hook:        _module.init,
		Owner:       moduleName,
		HookNum:     proc.ACTION_START,
		Priority:    proc.PRI_SYS_INIT,
		SubPriority: options.PRI_M_AUTHN - 1,
	}}
)

func (p *module) init(ctx context.Context) error {
	c := configer.ConfigerMustFrom(ctx)

	cf := session.NewConfig()
	if err := c.Read(p.name, cf); err != nil {
		return err
	}
	p.config = cf

	sm, err := startSession(cf, ctx)
	if err != nil {
		return err
	}

	options.WithSessionManager(ctx, sm)

	return nil
}

func startSession(cf *session.Config, ctx context.Context) (session.SessionManager, error) {
	opts := []session.Option{session.WithCtx(ctx)}
	if cf.Storage == "db" {
		db, ok := options.DBFrom(ctx, cf.DBName)
		if !ok {
			return nil, fmt.Errorf("can not found db[%s] from context", cf.DBName)
		}
		opts = append(opts, session.WithDB(db))
	}
	if cf.CookieName == session.DefCookieName {
		cf.CookieName = fmt.Sprintf("%s-sid", proc.Name())
	}
	return session.StartSession(cf, opts...)
}

func Register() {
	proc.RegisterHooks(hookOps)
	proc.RegisterFlags(moduleName, "session", session.NewConfig())
}
