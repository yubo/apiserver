package models

import (
	"context"
	"examples/rest/api"

	"github.com/yubo/apiserver/pkg/models"
	"github.com/yubo/apiserver/pkg/storage"
	"github.com/yubo/golib/orm"
)

func NewUser() *User {
	return &User{DB: models.DB()}
}

type User struct {
	orm.DB
}

func (p *User) Name() string {
	return "user"
}

func (p *User) NewObj() interface{} {
	return &api.User{}
}

func (p *User) Create(ctx context.Context, obj *api.User) error {
	return p.Insert(ctx, obj, orm.WithTable(p.Name()))
}

// Get retrieves the User from the db for a given name.
func (p *User) Get(ctx context.Context, name string) (ret *api.User, err error) {
	err = p.Query(ctx, "select * from user where name=?", name).Row(&ret)
	return
}

// List lists all Users in the indexer.
func (p *User) List(ctx context.Context, opts storage.ListOptions) (list []api.User, err error) {
	err = p.DB.List(
		ctx,
		&list,
		orm.WithTable(p.Name()),
		orm.WithTotal(opts.Total),
		orm.WithSelector(opts.Query),
		orm.WithOrderby(opts.Orderby...),
		orm.WithLimit(opts.Offset, opts.Limit),
	)
	return
}

func (p *User) Update(ctx context.Context, obj *api.User) error {
	return p.DB.Update(ctx, obj, orm.WithTable(p.Name()))
}

func (p *User) Delete(ctx context.Context, name string) error {
	_, err := p.DB.Exec(ctx, "delete from user where name=?", name)
	return err
}

func init() {
	models.Register(&User{})
}
