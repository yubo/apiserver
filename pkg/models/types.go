package models

import (
	"context"

	"github.com/yubo/apiserver/pkg/storage"
	"github.com/yubo/golib/runtime"
)

// store: kv store
// k8s.io/apiserver/pkg/registry/generic/registry/store.go
// k8s.io/apiserver/pkg/storage/interfaces.go
type ModelStore struct {
	store    storage.Store
	resource string
}

func (p ModelStore) Kind() string {
	return p.resource
}
func (p ModelStore) Create(ctx context.Context, name string, obj, out runtime.Object) error {
	return p.store.Create(ctx, p.resource+"/"+name, obj, out)
}
func (p ModelStore) Get(ctx context.Context, name string, ignoreNotFound bool, out runtime.Object) error {
	return p.store.Get(ctx, p.resource+"/"+name, storage.GetOptions{IgnoreNotFound: ignoreNotFound}, out)
}

func (p ModelStore) List(ctx context.Context, opts storage.ListOptions, out runtime.Object, count *int) error {
	return p.store.List(ctx, p.resource, opts, out, count)
}

func (p ModelStore) Update(ctx context.Context, name string, obj, out runtime.Object) error {
	return p.store.Update(ctx, p.resource+"/"+name, obj, out)
}

func (p ModelStore) Delete(ctx context.Context, name string, out runtime.Object) error {
	return p.store.Delete(ctx, p.resource+"/"+name, out)
}

type Model interface {
	Name() string
	NewObj() interface{}
}

type Models interface {
	Register(ms ...Model)
	NewModelStore(kind string) ModelStore
}
