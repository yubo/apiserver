package db

import (
	"github.com/yubo/apiserver/pkg/api/rbac"
	"github.com/yubo/apiserver/pkg/listers"
	"github.com/yubo/apiserver/pkg/storage"
	"github.com/yubo/golib/labels"
	"github.com/yubo/golib/orm"
)

// roleLister implements the RoleLister interface.
type roleLister struct {
	db *orm.DB
}

// NewRoleLister returns a new RoleLister.
func NewRoleLister(db *orm.DB) listers.RoleLister {
	return &roleLister{db: db}
}

// List lists all Roles in the indexer.
func (s *roleLister) List(selector labels.Selector) (ret []*rbac.Role, err error) {
	err = storage.List(s.db, "role", selector, &ret)
	return
}

// Get retrieves the Role from the db for a given name.
func (s *roleLister) Get(name string) (ret *rbac.Role, err error) {
	err = storage.Get(s.db, "role", name, &ret)
	return
}
