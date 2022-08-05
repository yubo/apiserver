// this is a sample echo rest api module
package user

import (
	"context"

	"github.com/yubo/apiserver/pkg/models"
	"github.com/yubo/apiserver/pkg/storage"
)

type UserModel interface {
	Name() string
	NewObj() interface{}

	List(ctx context.Context, opts storage.ListOptions) ([]*User, error)
	Get(ctx context.Context, name string) (*User, error)
	Create(ctx context.Context, obj *User) (*User, error)
	Update(ctx context.Context, obj *UpdateUserInput) (*User, error)
	Delete(ctx context.Context, name string) (*User, error)
}

func NewUser() UserModel {
	o := &user{}
	o.store = models.NewModelStore(o.Name())
	return o
}

// user implements the user interface.
type user struct {
	store models.ModelStore
}

func (p *user) Name() string {
	return "user"
}

func (p *user) NewObj() interface{} {
	return &User{}
}

func (p *user) Create(ctx context.Context, obj *User) (ret *User, err error) {
	err = p.store.Create(ctx, obj.Name, obj, &ret)
	return
}

// Get retrieves the User from the db for a given name.
func (p *user) Get(ctx context.Context, name string) (ret *User, err error) {
	err = p.store.Get(ctx, name, false, &ret)
	return
}

// List lists all Users in the indexer.
func (p *user) List(ctx context.Context, opts storage.ListOptions) (list []*User, err error) {
	err = p.store.List(ctx, opts, &list, opts.Total)
	return
}

func (p *user) Update(ctx context.Context, obj *UpdateUserInput) (ret *User, err error) {
	err = p.store.Update(ctx, obj.Name, obj, &ret)
	return
}

func (p *user) Delete(ctx context.Context, name string) (ret *User, err error) {
	err = p.store.Delete(ctx, name, &ret)
	return
}

func init() {
	models.Register(&user{})
}
