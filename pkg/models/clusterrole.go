package models

import (
	"context"

	"github.com/yubo/apiserver/pkg/apis/rbac"
	"github.com/yubo/apiserver/pkg/storage"
)

// ClusterRoleLister helps list RolesBinding.
// All objects returned here must be treated as read-only.
type ClusterRole interface {
	Name() string
	NewObj() interface{}

	List(ctx context.Context, opts storage.ListOptions) ([]*rbac.ClusterRole, error)
	Get(ctx context.Context, name string) (*rbac.ClusterRole, error)
	Create(ctx context.Context, obj *rbac.ClusterRole) (*rbac.ClusterRole, error)
	Update(ctx context.Context, obj *rbac.ClusterRole) (*rbac.ClusterRole, error)
	Delete(ctx context.Context, name string) (*rbac.ClusterRole, error)
}

// pkg/registry/rbac/role/storage/storage.go
// pkg/registry/rbac/rest/storage_rbac.go
func NewClusterRole() ClusterRole {
	o := &clusterRole{}
	o.store = NewStore(o.Name())
	return o
}

// clusterRole implements the role interface.
type clusterRole struct {
	store Store
}

func (p *clusterRole) Name() string {
	return "cluster_role"
}

func (p *clusterRole) NewObj() interface{} {
	return &rbac.ClusterRole{}
}

func (p *clusterRole) Create(ctx context.Context, obj *rbac.ClusterRole) (ret *rbac.ClusterRole, err error) {
	err = p.store.Create(ctx, obj.Name, obj, &ret)
	return
}

// Get retrieves the ClusterRole from the db for a given name.
func (p *clusterRole) Get(ctx context.Context, name string) (ret *rbac.ClusterRole, err error) {
	err = p.store.Get(ctx, name, false, &ret)
	return
}

// List lists all ClusterRoles in the indexer.
func (p *clusterRole) List(ctx context.Context, opts storage.ListOptions) (list []*rbac.ClusterRole, err error) {
	err = p.store.List(ctx, opts, &list, opts.Total)
	return
}

func (p *clusterRole) Update(ctx context.Context, obj *rbac.ClusterRole) (ret *rbac.ClusterRole, err error) {
	err = p.store.Update(ctx, obj.Name, obj, &ret)
	return
}

func (p *clusterRole) Delete(ctx context.Context, name string) (ret *rbac.ClusterRole, err error) {
	err = p.store.Delete(ctx, name, &ret)
	return
}

func init() {
	Register(&clusterRole{})
}
