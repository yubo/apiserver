package listers

import (
	"github.com/yubo/apiserver/pkg/storage"
	"github.com/yubo/apiserver/pkg/api/rbac"
	"github.com/yubo/golib/labels"
	"github.com/yubo/golib/orm"
)

// RoleLister helps list Roles.
// All objects returned here must be treated as read-only.
type RoleLister interface {
	// List lists all Roles in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*rbac.Role, err error)
	Get(name string) (*rbac.Role, error)
}

// roleLister implements the RoleLister interface.
type roleLister struct {
	db *orm.DB
}

// NewRoleLister returns a new RoleLister.
func NewRoleLister(db *orm.DB) RoleLister {
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
