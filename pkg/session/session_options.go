package session

import (
	"context"

	"github.com/yubo/apiserver/pkg/session/types"
	"github.com/yubo/golib/util/clock"
)

type Options struct {
	ctx      context.Context
	cancel   context.CancelFunc
	clock    clock.Clock
	sessions types.Session
}

type Option func(*Options)

func WithCtx(ctx context.Context) Option {
	return func(o *Options) {
		o.ctx = ctx
		o.cancel = nil
	}
}

func WithModel(m types.Session) Option {
	return func(o *Options) {
		o.sessions = m
	}
}

func WithClock(clock clock.Clock) Option {
	return func(o *Options) {
		o.clock = clock
	}
}
