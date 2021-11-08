package register

import (
	"context"
	"fmt"

	"github.com/yubo/apiserver/pkg/options"
	"github.com/yubo/golib/net/session"
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

func newConfig() *session.Config {
	return &session.Config{
		SidLength:      24,
		HttpOnly:       true,
		GcInterval:     60,
		CookieLifetime: 16 * 3600,
	}
}

func (p *module) init(ctx context.Context) error {
	c := proc.ConfigerMustFrom(ctx)

	cf := newConfig()
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
	if cf.Storage == "db" && cf.Dsn == "" {
		db, ok := options.DBFrom(ctx, "")
		if !ok {
			return nil, fmt.Errorf("can not found db from context")
		}
		opts = append(opts, session.WithDB(db))
	}
	if cf.CookieName == session.DefCookieName {
		cf.CookieName = fmt.Sprintf("%s-sid", proc.Name())
	}
	return session.StartSession(cf, opts...)
}
