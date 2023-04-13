package models

import (
	"context"

	"examples/models-kv/api"

	"github.com/yubo/apiserver/pkg/models"
	libapi "github.com/yubo/golib/api"
)

// SecretLister helps list RolesBinding.
// All objects returned here must be treated as read-only.
type Demo interface {
	Name() string
	NewObj() interface{}

	List(ctx context.Context, opts libapi.GetListOptions) ([]*api.Demo, error)
	Get(ctx context.Context, name string) (*api.Demo, error)
	Create(ctx context.Context, obj *api.Demo) (*api.Demo, error)
	Update(ctx context.Context, obj *api.Demo) (*api.Demo, error)
	Delete(ctx context.Context, name string) (*api.Demo, error)
}

// pkg/registry/rbac/role/storage/storage.go
// pkg/registry/rbac/rest/storage_rbac.go
func NewDemo() Demo {
	o := &demo{}
	o.store = models.NewModelStore(o.Name())
	return o
}

// demo implements the role interface.
type demo struct {
	store models.ModelStore
}

func (p *demo) Name() string {
	return "demo"
}

func (p *demo) NewObj() interface{} {
	return &api.Demo{}
}

func (p *demo) Create(ctx context.Context, obj *api.Demo) (ret *api.Demo, err error) {
	err = p.store.Create(ctx, obj.Name, obj, &ret)
	return
}

// Get retrieves the Secret from the db for a given name.
func (p *demo) Get(ctx context.Context, name string) (ret *api.Demo, err error) {
	err = p.store.Get(ctx, name, false, &ret)
	return
}

// List lists all Secrets in the indexer.
func (p *demo) List(ctx context.Context, opts libapi.GetListOptions) (list []*api.Demo, err error) {
	err = p.store.List(ctx, opts, &list, opts.Total)
	return
}

func (p *demo) Update(ctx context.Context, obj *api.Demo) (ret *api.Demo, err error) {
	err = p.store.Update(ctx, obj.Name, obj, &ret)
	return
}

func (p *demo) Delete(ctx context.Context, name string) (ret *api.Demo, err error) {
	err = p.store.Delete(ctx, name, &ret)
	return
}

func init() {
	models.Register(&demo{})
}
