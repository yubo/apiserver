package models

import (
	"context"

	"github.com/yubo/apiserver/pkg/apis/rbac"
	"github.com/yubo/apiserver/pkg/storage"
)

type Role interface {
	Name() string
	NewObj() interface{}

	List(ctx context.Context, opts storage.ListOptions) (total int64, list []*rbac.Role, err error)
	Get(ctx context.Context, name string) (*rbac.Role, error)
	Create(ctx context.Context, obj *rbac.Role) (*rbac.Role, error)
	Update(ctx context.Context, obj *rbac.Role) (*rbac.Role, error)
	Delete(ctx context.Context, name string) (*rbac.Role, error)
}

// pkg/registry/rbac/role/storage/storage.go
// pkg/registry/rbac/rest/storage_rbac.go
func NewRole() Role {
	return &role{store: NewStore("role")}
}

// role implements the role interface.
type role struct {
	store Store
}

func (p *role) Name() string {
	return "role"
}

func (p *role) NewObj() interface{} {
	return &rbac.Role{}
}

func (p *role) Create(ctx context.Context, obj *rbac.Role) (ret *rbac.Role, err error) {
	err = p.store.Create(ctx, obj.Name, obj, &ret)
	return
}

// Get retrieves the Role from the db for a given name.
func (p *role) Get(ctx context.Context, name string) (ret *rbac.Role, err error) {
	err = p.store.Get(ctx, name, false, &ret)
	return
}

// List lists all Roles in the indexer.
func (p *role) List(ctx context.Context, opts storage.ListOptions) (total int64, list []*rbac.Role, err error) {
	err = p.store.List(ctx, opts, &list, &total)
	return
}

func (p *role) Update(ctx context.Context, obj *rbac.Role) (ret *rbac.Role, err error) {
	err = p.store.Update(ctx, obj.Name, obj, &ret)
	return
}

func (p *role) Delete(ctx context.Context, name string) (ret *rbac.Role, err error) {
	err = p.store.Delete(ctx, name, &ret)
	return
}

func init() {
	Register(&role{})
}
