package models

import (
	"context"

	"github.com/yubo/apiserver/pkg/storage"
	"github.com/yubo/golib/runtime"
)

// store: kv store
// k8s.io/apiserver/pkg/registry/generic/registry/store.go
// k8s.io/apiserver/pkg/storage/interfaces.go
type Store struct {
	kv       storage.KV
	resource string
}

func (p Store) Kind() string {
	return p.resource
}
func (p Store) Create(ctx context.Context, name string, obj, out runtime.Object) error {
	return p.kv.Create(ctx, p.resource+"/"+name, obj, out)
}
func (p Store) Get(ctx context.Context, name string, ignoreNotFound bool, out runtime.Object) error {
	return p.kv.Get(ctx, p.resource+"/"+name, storage.GetOptions{IgnoreNotFound: ignoreNotFound}, out)
}

func (p Store) List(ctx context.Context, opts storage.ListOptions, out runtime.Object, count *int64) error {
	return p.kv.List(ctx, p.resource, opts, out, count)
}

func (p Store) Update(ctx context.Context, name string, obj, out runtime.Object) error {
	return p.kv.Update(ctx, p.resource+"/"+name, obj, out)
}

func (p Store) Delete(ctx context.Context, name string, out runtime.Object) error {
	return p.kv.Delete(ctx, p.resource+"/"+name, out)
}

//func (p Store) Drop() error {
//	return p.s.Drop(p.resource)
//}

type Model interface {
	Name() string
	NewObj() interface{}
}
