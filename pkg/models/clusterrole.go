package models

import (
	"context"

	"github.com/yubo/apiserver/pkg/apis/rbac"
	"github.com/yubo/golib/api"
	"github.com/yubo/golib/orm"
)

// pkg/registry/rbac/role/storage/storage.go
// pkg/registry/rbac/rest/storage_rbac.go
func NewClusterRole() *clusterRole {
	return &clusterRole{DB: DB()}
}

// clusterRole implements the role interface.
type clusterRole struct {
	orm.DB
}

func (p *clusterRole) Name() string {
	return "cluster_role"
}

func (p *clusterRole) NewObj() interface{} {
	return &rbac.ClusterRole{}
}

func (p *clusterRole) Create(ctx context.Context, obj *rbac.ClusterRole) error {
	return p.Insert(ctx, obj)
}

// Get retrieves the ClusterRole from the db for a given name.
func (p *clusterRole) Get(ctx context.Context, name string) (ret *rbac.ClusterRole, err error) {
	err = p.Query(ctx, "select * from cluster_role where name=?", name).Row(&ret)
	return
}

// List lists all ClusterRoles in the indexer.
func (p *clusterRole) List(ctx context.Context, o api.GetListOptions) (list []*rbac.ClusterRole, err error) {
	err = p.DB.List(ctx, &list,
		orm.WithTable(p.Name()),
		orm.WithTotal(o.Total),
		orm.WithSelector(o.Query),
		orm.WithOrderby(o.Orderby...),
		orm.WithLimit(o.Offset, o.Limit),
	)
	return
}

func (p *clusterRole) Update(ctx context.Context, obj *rbac.ClusterRole) error {
	return p.DB.Update(ctx, obj)
}

func (p *clusterRole) Delete(ctx context.Context, name string) error {
	_, err := p.Exec(ctx, "delete cluster_role role where name=?", name)
	return err
}

func init() {
	Register(&clusterRole{})
}
