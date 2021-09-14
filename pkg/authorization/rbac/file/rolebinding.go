package file

import (
	"sort"

	"github.com/yubo/apiserver/pkg/api/rbac"
	"github.com/yubo/apiserver/pkg/listers"
	"github.com/yubo/golib/api/errors"
	"github.com/yubo/golib/labels"
)

// roleBindingLister implements the RoleBindingLister interface.
type roleBindingLister struct {
	*FileStorage
}

// NewRoleBindingLister returns a new RoleBindingLister.
func NewRoleBindingLister(f *FileStorage) listers.RoleBindingLister {
	return &roleBindingLister{FileStorage: f}
}

// List lists all RoleBindings in the indexer.
func (p *roleBindingLister) List(selector labels.Selector) (ret []*rbac.RoleBinding, err error) {
	return p.roleBindings, nil
}

// Get retrieves the RoleBinding from the db for a given name.
func (p *roleBindingLister) Get(name string) (ret *rbac.RoleBinding, err error) {
	a := p.roleBindings
	i := sort.Search(len(a), func(i int) bool { return a[i].Name >= name })
	if ret = a[i]; ret.Name == name {
		return ret, nil
	}

	return nil, errors.NewNotFound("RoleBinding")
}
