package models

import (
	"context"

	"github.com/yubo/apiserver/pkg/storage"
	"github.com/yubo/golib/api"
)

// SecretLister helps list RolesBinding.
// All objects returned here must be treated as read-only.
type Secret interface {
	Name() string
	NewObj() interface{}

	List(ctx context.Context, opts storage.ListOptions) (total int64, list []*api.Secret, err error)
	Get(ctx context.Context, name string) (*api.Secret, error)
	Create(ctx context.Context, obj *api.Secret) (*api.Secret, error)
	Update(ctx context.Context, obj *api.Secret) (*api.Secret, error)
	Delete(ctx context.Context, name string) (*api.Secret, error)
}

// pkg/registry/rbac/role/storage/storage.go
// pkg/registry/rbac/rest/storage_rbac.go
func NewSecret() Secret {
	return &secret{store: NewStore("secret")}
}

// secret implements the role interface.
type secret struct {
	store Store
}

func (p *secret) Name() string {
	return "secret"
}

func (p *secret) NewObj() interface{} {
	return &api.Secret{}
}

func (p *secret) Create(ctx context.Context, obj *api.Secret) (ret *api.Secret, err error) {
	err = p.store.Create(ctx, obj.Name, obj, &ret)
	return
}

// Get retrieves the Secret from the db for a given name.
func (p *secret) Get(ctx context.Context, name string) (ret *api.Secret, err error) {
	err = p.store.Get(ctx, name, false, &ret)
	return
}

// List lists all Secrets in the indexer.
func (p *secret) List(ctx context.Context, opts storage.ListOptions) (total int64, list []*api.Secret, err error) {
	err = p.store.List(ctx, opts, &list, &total)
	return
}

func (p *secret) Update(ctx context.Context, obj *api.Secret) (ret *api.Secret, err error) {
	err = p.store.Update(ctx, obj.Name, obj, &ret)
	return
}

func (p *secret) Delete(ctx context.Context, name string) (ret *api.Secret, err error) {
	err = p.store.Delete(ctx, name, &ret)
	return
}

func init() {
	Register(&secret{})
}
