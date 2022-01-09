package models

import (
	"context"

	"github.com/yubo/apiserver/pkg/apis/rbac"
	"github.com/yubo/apiserver/pkg/storage"
)

// RoleBindingLister helps list RolesBinding.
// All objects returned here must be treated as read-only.
type RoleBinding interface {
	Name() string
	NewObj() interface{}

	List(ctx context.Context, opts storage.ListOptions) ([]*rbac.RoleBinding, error)
	Get(ctx context.Context, name string) (*rbac.RoleBinding, error)
	Create(ctx context.Context, obj *rbac.RoleBinding) (*rbac.RoleBinding, error)
	Update(ctx context.Context, obj *rbac.RoleBinding) (*rbac.RoleBinding, error)
	Delete(ctx context.Context, name string) (*rbac.RoleBinding, error)
}

// pkg/registry/rbac/role/storage/storage.go
// pkg/registry/rbac/rest/storage_rbac.go
func NewRoleBinding() RoleBinding {
	o := &roleBinding{}
	o.store = NewStore(o.Name())
	return o
}

// roleBinding implements the role interface.
type roleBinding struct {
	store Store
}

func (p *roleBinding) Name() string {
	return "role_binding"
}

func (p *roleBinding) NewObj() interface{} {
	return &rbac.RoleBinding{}
}

func (p *roleBinding) Create(ctx context.Context, obj *rbac.RoleBinding) (ret *rbac.RoleBinding, err error) {
	err = p.store.Create(ctx, obj.Name, obj, &ret)
	return
}

// Get retrieves the RoleBinding from the db for a given name.
func (p *roleBinding) Get(ctx context.Context, name string) (ret *rbac.RoleBinding, err error) {
	err = p.store.Get(ctx, name, false, &ret)
	return
}

// List lists all RoleBindings in the indexer.
func (p *roleBinding) List(ctx context.Context, opts storage.ListOptions) (list []*rbac.RoleBinding, err error) {
	err = p.store.List(ctx, opts, &list, opts.Total)
	return
}

func (p *roleBinding) Update(ctx context.Context, obj *rbac.RoleBinding) (ret *rbac.RoleBinding, err error) {
	err = p.store.Update(ctx, obj.Name, obj, &ret)
	return
}

func (p *roleBinding) Delete(ctx context.Context, name string) (ret *rbac.RoleBinding, err error) {
	err = p.store.Delete(ctx, name, &ret)
	return
}

func init() {
	Register(&roleBinding{})
}
