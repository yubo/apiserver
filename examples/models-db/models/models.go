package models

import (
	"context"

	"examples/models/api"

	"github.com/yubo/apiserver/pkg/models"
	"github.com/yubo/apiserver/pkg/storage"
	"github.com/yubo/golib/orm"
)

// pkg/registry/rbac/role/storage/storage.go
// pkg/registry/rbac/rest/storage_rbac.go
func NewDemo() *Demo {
	return &Demo{DB: models.DB()}
}

// demo implements the role interface.
type Demo struct {
	orm.DB
}

func (p *Demo) Name() string {
	return "demo"
}

func (p *Demo) NewObj() interface{} {
	return &api.Demo{}
}

func (p *Demo) Create(ctx context.Context, obj *api.Demo) error {
	return p.DB.Insert(obj, orm.WithTable(p.Name()))
}

// Get retrieves the Secret from the db for a given name.
func (p *Demo) Get(ctx context.Context, selector string) (ret *api.Demo, err error) {
	err = p.DB.Get(&ret, orm.WithTable(p.Name()), orm.WithSelector(selector))
	return
}

// List lists all Secrets in the indexer.
func (p *Demo) List(ctx context.Context, opts storage.ListOptions) (list []*api.Demo, err error) {
	err = p.DB.List(&list,
		orm.WithTable(p.Name()),
		orm.WithTotal(opts.Total),
		orm.WithSelector(opts.Query),
		orm.WithOrderby(opts.Orderby...),
		orm.WithLimit(opts.Offset, opts.Limit),
	)
	return
}

func (p *Demo) Update(ctx context.Context, obj *api.Demo) error {
	return p.DB.Update(obj, orm.WithTable(p.Name()))
}

func (p *Demo) Delete(ctx context.Context, selector string) error {
	return p.DB.Delete(nil, orm.WithTable(p.Name()), orm.WithSelector(selector))
}

func init() {
	models.Register(&Demo{})
}
