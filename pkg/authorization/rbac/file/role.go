package file

import (
	"sort"

	"github.com/yubo/apiserver/pkg/api/rbac"
	"github.com/yubo/apiserver/pkg/listers"
	"github.com/yubo/golib/api/errors"
	"github.com/yubo/golib/labels"
)

// roleLister implements the RoleLister interface.
type roleLister struct {
	*FileStorage
}

// NewRoleLister returns a new RoleLister.
func NewRoleLister(f *FileStorage) listers.RoleLister {
	return &roleLister{FileStorage: f}
}

// List lists all Roles in the indexer.
func (p *roleLister) List(selector labels.Selector) (ret []*rbac.Role, err error) {
	return p.roles, nil
}

// Get retrieves the Role from the db for a given name.
func (p *roleLister) Get(name string) (ret *rbac.Role, err error) {
	a := p.roles
	if i := sort.Search(len(a), func(i int) bool { return a[i].Name >= name }); i < len(a) {
		if ret = a[i]; ret.Name == name {
			return ret, nil
		}
	}

	return nil, errors.NewNotFound("Role")
}
