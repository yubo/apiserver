package models

import (
	"context"

	"github.com/yubo/apiserver/pkg/apis/rbac"
	"github.com/yubo/golib/api"
	"github.com/yubo/golib/orm"
)

// pkg/registry/rbac/role/storage/storage.go
// pkg/registry/rbac/rest/storage_rbac.go
func NewRoleBinding() *RoleBinding {
	return &RoleBinding{DB: DB()}
}

// RoleBinding implements the role interface.
type RoleBinding struct {
	orm.DB
}

func (p *RoleBinding) Name() string {
	return "role_binding"
}

func (p *RoleBinding) NewObj() interface{} {
	return &rbac.RoleBinding{}
}

func (p *RoleBinding) Create(ctx context.Context, obj *rbac.RoleBinding) error {
	return p.Insert(ctx, obj)
}

// Get retrieves the RoleBinding from the db for a given name.
func (p *RoleBinding) Get(ctx context.Context, name string) (ret *rbac.RoleBinding, err error) {
	err = p.Query(ctx, "select * from role_binding where name=?", name).Row(&ret)
	return
}

// List lists all RoleBindings in the indexer.
func (p *RoleBinding) List(ctx context.Context, o api.GetListOptions) (list []*rbac.RoleBinding, err error) {
	err = p.DB.List(ctx, &list,
		orm.WithTable(p.Name()),
		orm.WithTotal(o.Total),
		orm.WithSelector(o.Query),
		orm.WithOrderby(o.Orderby...),
		orm.WithLimit(o.Offset, o.Limit),
	)
	return
}

func (p *RoleBinding) Update(ctx context.Context, obj *rbac.RoleBinding) error {
	return p.DB.Update(ctx, obj)
}

func (p *RoleBinding) Delete(ctx context.Context, name string) error {
	_, err := p.Exec(ctx, "delete from role_binding where name=?", name)
	return err
}

func init() {
	Register(&RoleBinding{})
}
