package mem

import (
	"context"

	"github.com/yubo/apiserver/pkg/storage"
	"github.com/yubo/golib/api"
	"github.com/yubo/golib/runtime"
)

var _ storage.Store = &store{}

type store struct{}

func New() storage.Store {
	return newStore()
}

func newStore() *store {
	return &store{}
}

func (p store) Create(ctx context.Context, key string, obj, out runtime.Object) error {
	return nil
}

func (p store) Delete(ctx context.Context, key string, out runtime.Object) error {
	return nil
}

func (p store) Update(ctx context.Context, key string, obj, out runtime.Object) error {
	return nil
}

func (p store) Get(ctx context.Context, key string, opts api.GetOptions, out runtime.Object) error {
	return nil
}

func (p store) List(ctx context.Context, key string, opts api.GetListOptions, out runtime.Object, total *int) error {
	return nil
}
