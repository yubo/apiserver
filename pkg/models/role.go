package models

import (
	"context"

	"github.com/yubo/apiserver/pkg/apis/rbac"
	"github.com/yubo/apiserver/pkg/storage"
	"github.com/yubo/golib/orm"
)

// pkg/registry/rbac/role/storage/storage.go
// pkg/registry/rbac/rest/storage_rbac.go
func NewRole() *Role {
	return &Role{DB: DB()}
}

// Role implements the Role interface.
type Role struct {
	orm.DB
}

func (p *Role) Name() string {
	return "role"
}

func (p *Role) NewObj() interface{} {
	return &rbac.Role{}
}

func (p *Role) Create(ctx context.Context, obj *rbac.Role) error {
	return p.Insert(ctx, obj)
}

// Get retrieves the Role from the db for a given name.
func (p *Role) Get(ctx context.Context, name string) (ret *rbac.Role, err error) {
	err = p.Query(ctx, "select * from role where name=?", name).Row(&ret)
	return
}

// List lists all Roles in the indexer.
func (p *Role) List(ctx context.Context, o storage.ListOptions) (list []*rbac.Role, err error) {
	err = p.DB.List(ctx, &list,
		orm.WithTable(p.Name()),
		orm.WithTotal(o.Total),
		orm.WithSelector(o.Query),
		orm.WithOrderby(o.Orderby...),
		orm.WithLimit(o.Offset, o.Limit),
	)
	return
}

func (p *Role) Update(ctx context.Context, obj *rbac.Role) error {
	return p.DB.Update(ctx, obj)
}

func (p *Role) Delete(ctx context.Context, name string) error {
	_, err := p.Exec(ctx, "delete from role where name=?", name)
	return err
}

func init() {
	Register(&Role{})
}
