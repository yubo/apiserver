package register

import (
	"context"
	"fmt"
	"time"

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
		HookNum:     proc.ACTION_TEST,
		Priority:    proc.PRI_SYS_INIT,
		SubPriority: options.PRI_M_SESSION,
	}, {
		Hook:        _module.init,
		Owner:       moduleName,
		HookNum:     proc.ACTION_START,
		Priority:    proc.PRI_SYS_INIT,
		SubPriority: options.PRI_M_SESSION,
	}}
)

func newConfig() *session.Config {
	return &session.Config{
		SidLength:      24,
		HttpOnly:       true,
		GcInterval:     60 * time.Second,
		CookieLifetime: 16 * time.Hour,
	}
}

func startSession(cf *session.Config, ctx context.Context) (session.SessionManager, error) {
	opts := []session.Option{session.WithCtx(ctx)}
	if cf.Storage == "db" && cf.Dsn == "" {
		db, ok := options.DBFrom(ctx)
		if !ok {
			return nil, fmt.Errorf("can not found db from context")
		}
		opts = append(opts, session.WithDB(db))
	}
	if cf.CookieName == session.DefCookieName {
		cf.CookieName = fmt.Sprintf("%s-sid", proc.NameFrom(ctx))
	}
	return session.StartSession(cf, opts...)
}

func (p *module) init(ops *proc.HookOps) error {
	ctx, configer := ops.ContextAndConfiger()

	cf := newConfig()
	if err := configer.ReadYaml(p.name, cf); err != nil {
		return err
	}
	p.config = cf

	sm, err := startSession(cf, ctx)
	if err != nil {
		return err
	}

	ops.SetContext(options.WithSessionManager(ctx, sm))

	return nil
}
