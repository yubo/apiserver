package allinone

import (
	"context"

	"examples/all-in-one/pkg/allinone/authn"
	"examples/all-in-one/pkg/allinone/authz"
	"examples/all-in-one/pkg/allinone/config"
	"examples/all-in-one/pkg/allinone/session"
	"examples/all-in-one/pkg/allinone/trace"
	"examples/all-in-one/pkg/allinone/user"

	"github.com/yubo/golib/proc"
)

type allinone struct {
	ctx context.Context
}

func New() *allinone {
	return &allinone{}
}

func (p *allinone) Start(ctx context.Context) error {
	cf := config.New()
	if err := proc.ReadConfig("allinone", cf); err != nil {
		return err
	}

	session.New(ctx, cf).Install()
	trace.New(ctx, cf).Install()
	user.New(ctx, cf).Install()
	authn.New(ctx, cf).Install()
	authz.New(ctx, cf).Install()

	return nil
}
