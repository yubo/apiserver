package session

import (
	"context"

	"github.com/yubo/golib/orm"
	"github.com/yubo/golib/util/clock"
)

type Options struct {
	ctx    context.Context
	cancel context.CancelFunc
	clock  clock.Clock
	db     orm.DB
	mem    bool
}

type Option func(*Options)

func WithCtx(ctx context.Context) Option {
	return func(o *Options) {
		o.ctx = ctx
		o.cancel = nil
	}
}

func WithDB(db orm.DB) Option {
	return func(o *Options) {
		o.db = db
	}
}

func WithClock(clock clock.Clock) Option {
	return func(o *Options) {
		o.clock = clock
	}
}

func WithMem() Option {
	return func(o *Options) {
		o.mem = true
	}
}
