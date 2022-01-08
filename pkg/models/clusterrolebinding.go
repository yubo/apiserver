package models

import (
	"context"

	"github.com/yubo/apiserver/pkg/apis/rbac"
	"github.com/yubo/apiserver/pkg/storage"
)

// ClusterRoleBindingLister helps list RolesBinding.
// All objects returned here must be treated as read-only.
type ClusterRoleBinding interface {
	Name() string
	NewObj() interface{}

	List(ctx context.Context, opts storage.ListOptions) (total int64, list []*rbac.ClusterRoleBinding, err error)
	Get(ctx context.Context, name string) (*rbac.ClusterRoleBinding, error)
	Create(ctx context.Context, obj *rbac.ClusterRoleBinding) (*rbac.ClusterRoleBinding, error)
	Update(ctx context.Context, obj *rbac.ClusterRoleBinding) (*rbac.ClusterRoleBinding, error)
	Delete(ctx context.Context, name string) (*rbac.ClusterRoleBinding, error)
}

// pkg/registry/rbac/role/storage/storage.go
// pkg/registry/rbac/rest/storage_rbac.go
func NewClusterRoleBinding() ClusterRoleBinding {
	return &clusterRoleBinding{store: NewStore("role_binding")}
}

// clusterRoleBinding implements the role interface.
type clusterRoleBinding struct {
	store Store
}

func (p *clusterRoleBinding) Name() string {
	return "role_binding"
}

func (p *clusterRoleBinding) NewObj() interface{} {
	return &rbac.ClusterRoleBinding{}
}

func (p *clusterRoleBinding) Create(ctx context.Context, obj *rbac.ClusterRoleBinding) (ret *rbac.ClusterRoleBinding, err error) {
	err = p.store.Create(ctx, obj.Name, obj, &ret)
	return
}

// Get retrieves the ClusterRoleBinding from the db for a given name.
func (p *clusterRoleBinding) Get(ctx context.Context, name string) (ret *rbac.ClusterRoleBinding, err error) {
	err = p.store.Get(ctx, name, false, &ret)
	return
}

// List lists all ClusterRoleBindings in the indexer.
func (p *clusterRoleBinding) List(ctx context.Context, opts storage.ListOptions) (total int64, list []*rbac.ClusterRoleBinding, err error) {
	err = p.store.List(ctx, opts, &list, &total)
	return
}

func (p *clusterRoleBinding) Update(ctx context.Context, obj *rbac.ClusterRoleBinding) (ret *rbac.ClusterRoleBinding, err error) {
	err = p.store.Update(ctx, obj.Name, obj, &ret)
	return
}

func (p *clusterRoleBinding) Delete(ctx context.Context, name string) (ret *rbac.ClusterRoleBinding, err error) {
	err = p.store.Delete(ctx, name, &ret)
	return
}

func init() {
	Register(&clusterRoleBinding{})
}
