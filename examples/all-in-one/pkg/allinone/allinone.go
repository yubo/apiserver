package allinone

import (
	"context"

	"examples/all-in-one/pkg/allinone/authn"
	"examples/all-in-one/pkg/allinone/authz"
	"examples/all-in-one/pkg/allinone/session"
	"examples/all-in-one/pkg/allinone/trace"
	"examples/all-in-one/pkg/allinone/user"
)

type allinone struct {
	ctx context.Context
}

func New(ctx context.Context) *allinone {
	return &allinone{ctx: ctx}
}

func (p *allinone) Start() error {
	if err := session.New(p.ctx).Start(); err != nil {
		return err
	}
	if err := trace.New(p.ctx).Start(); err != nil {
		return err
	}
	if err := user.New(p.ctx).Start(); err != nil {
		return err
	}
	if err := authn.New(p.ctx).Start(); err != nil {
		return err
	}
	if err := authz.New(p.ctx).Start(); err != nil {
		return err
	}

	return nil
}
