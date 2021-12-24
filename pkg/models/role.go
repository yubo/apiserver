package models

import (
	"github.com/yubo/apiserver/pkg/apis/rbac"
	"github.com/yubo/golib/queries"
)

// RoleLister helps list Roles.
// All objects returned here must be treated as read-only.
type RoleLister interface {
	// List lists all Roles in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector queries.Selector) (ret []*rbac.Role, err error)
	Get(name string) (*rbac.Role, error)
}

// role implements the Role interface.
type role struct {
	baseModel
}

// NewRoleLister returns a new RoleLister.
func NewRoleLister() RoleLister {
	return &role{baseModel{kind: "role"}}
}

// List lists all Roles in the indexer.
func (p *role) List(selector queries.Selector) (ret []*rbac.Role, err error) {
	err = p.list(selector, &ret)
	return
}

// Get retrieves the Role from the db for a given name.
func (p *role) Get(name string) (ret *rbac.Role, err error) {
	err = p.get(name, &ret)
	return
}

func (p *role) Create(obj *rbac.Role) (*rbac.Role, error) {
	if err := p.create(obj); err != nil {
		return nil, err
	}

	return p.Get(obj.Name)
}

func (p *role) Update(obj *rbac.Role) (*rbac.Role, error) {
	if err := p.update(obj); err != nil {
		return nil, err
	}

	return p.Get(obj.Name)
}

func (p *role) Delete(obj *rbac.Role) (*rbac.Role, error) {
	ret, err := p.Get(obj.Name)
	if err != nil {
		return nil, err
	}

	if err := p.delete(obj); err != nil {
		return nil, err
	}

	return ret, nil
}
