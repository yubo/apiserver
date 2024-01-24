package models

import (
	"context"

	"github.com/yubo/apiserver/pkg/db"
	"github.com/yubo/golib/api"
	"github.com/yubo/golib/orm"
)

// depended by
//   plugin/authenticator/token/bootstrap
func NewSecret() *Secret {
	return &Secret{DB: DB()}
}

// Secret implements the role interface.
type Secret struct {
	orm.DB
}

func (p *Secret) Name() string {
	return "secret"
}

func (p *Secret) NewObj() interface{} {
	return &api.Secret{}
}

func (p *Secret) Create(ctx context.Context, obj *api.Secret) (err error) {
	return p.Insert(ctx, obj)
}

// Get retrieves the Secret from the db for a given name.
func (p *Secret) Get(ctx context.Context, name string) (ret *api.Secret, err error) {
	err = p.Query(ctx, "select * from secret where name=?", name).Row(&ret)
	return
}

// List lists all Secrets in the indexer.
func (p *Secret) List(ctx context.Context, o api.GetListOptions) (list []*api.Secret, err error) {
	err = p.DB.List(ctx, &list,
		orm.WithTable(p.Name()),
		orm.WithTotal(o.Total),
		orm.WithSelector(o.Query),
		orm.WithOrderby(o.Orderby...),
		orm.WithLimit(o.Offset, o.Limit),
	)
	return
}

func (p *Secret) Update(ctx context.Context, obj *api.Secret) error {
	return p.DB.Update(ctx, obj)
}

func (p *Secret) Delete(ctx context.Context, name string) error {
	_, err := p.Exec(ctx, "delete from secret where name=?", name)
	return err
}

func init() {
	db.Models(&Secret{})
}
