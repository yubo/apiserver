package models

import (
	"context"

	"examples/rest/api"

	"github.com/yubo/apiserver/pkg/models"
	"github.com/yubo/apiserver/pkg/storage"
)

type User interface {
	Name() string
	NewObj() interface{}

	List(ctx context.Context, opts storage.ListOptions) ([]api.User, error)
	Get(ctx context.Context, name string) (*api.User, error)
	Create(ctx context.Context, obj *api.User) (*api.User, error)
	Update(ctx context.Context, obj *api.UpdateUserInput) (*api.User, error)
	Delete(ctx context.Context, name string) (*api.User, error)
}

func NewUser() User {
	o := &user{}
	o.store = models.NewStore(o.Name())
	return o
}

// user implements the user interface.
type user struct {
	store models.Store
}

func (p *user) Name() string {
	return "user"
}

func (p *user) NewObj() interface{} {
	return &api.User{}
}

func (p *user) Create(ctx context.Context, obj *api.User) (ret *api.User, err error) {
	err = p.store.Create(ctx, obj.Name, obj, &ret)
	return
}

// Get retrieves the User from the db for a given name.
func (p *user) Get(ctx context.Context, name string) (ret *api.User, err error) {
	err = p.store.Get(ctx, name, false, &ret)
	return
}

// List lists all Users in the indexer.
func (p *user) List(ctx context.Context, opts storage.ListOptions) (list []api.User, err error) {
	err = p.store.List(ctx, opts, &list, opts.Total)
	return
}

func (p *user) Update(ctx context.Context, obj *api.UpdateUserInput) (ret *api.User, err error) {
	err = p.store.Update(ctx, obj.Name, obj, &ret)
	return
}

func (p *user) Delete(ctx context.Context, name string) (ret *api.User, err error) {
	err = p.store.Delete(ctx, name, &ret)
	return
}

func init() {
	models.Register(&user{})
}
