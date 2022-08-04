package file

import (
	"context"

	"github.com/yubo/apiserver/pkg/storage"
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

func (p store) Get(ctx context.Context, key string, opts storage.GetOptions, out runtime.Object) error {
	return nil
}

func (p store) List(ctx context.Context, key string, opts storage.ListOptions, out runtime.Object, total *int64) error {
	return nil
}
