package models

import (
	"context"

	"github.com/yubo/apiserver/pkg/apis/rbac"
	"github.com/yubo/apiserver/pkg/storage"
	"github.com/yubo/golib/orm"
)

func NewClusterRoleBinding() *ClusterRoleBinding {
	return &ClusterRoleBinding{DB: DB()}
}

// ClusterRoleBinding implements the role interface.
type ClusterRoleBinding struct {
	orm.DB
}

func (p *ClusterRoleBinding) Name() string {
	return "cluster_role_binding"
}

func (p *ClusterRoleBinding) NewObj() interface{} {
	return &rbac.ClusterRoleBinding{}
}

func (p *ClusterRoleBinding) Create(ctx context.Context, obj *rbac.ClusterRoleBinding) error {
	return p.Insert(ctx, obj)
}

// Get retrieves the ClusterRoleBinding from the db for a given name.
func (p *ClusterRoleBinding) Get(ctx context.Context, name string) (ret *rbac.ClusterRoleBinding, err error) {
	err = p.Query(ctx, "select * from cluster_role_binding where name=?", name).Row(&ret)
	return
}

// List lists all ClusterRoleBindings in the indexer.
func (p *ClusterRoleBinding) List(ctx context.Context, o storage.ListOptions) (list []*rbac.ClusterRoleBinding, err error) {
	err = p.DB.List(ctx, &list,
		orm.WithTable(p.Name()),
		orm.WithTotal(o.Total),
		orm.WithSelector(o.Query),
		orm.WithOrderby(o.Orderby...),
		orm.WithLimit(o.Offset, o.Limit),
	)
	return
}

func (p *ClusterRoleBinding) Update(ctx context.Context, obj *rbac.ClusterRoleBinding) error {
	return p.DB.Update(ctx, obj)
}

func (p *ClusterRoleBinding) Delete(ctx context.Context, name string) error {
	_, err := p.Exec(ctx, "delete from cluster_role_binding where name=?", name)
	return err
}

func init() {
	Register(&ClusterRoleBinding{})
}
