package file

import (
	"context"
	"sort"

	"github.com/yubo/apiserver/pkg/apis/rbac"
	"github.com/yubo/apiserver/pkg/listers"
	"github.com/yubo/golib/api"
	"github.com/yubo/golib/api/errors"
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
func (p *roleBindingLister) List(ctx context.Context, opts api.GetListOptions) (ret []*rbac.RoleBinding, err error) {
	return p.roleBindings, nil
}

// Get retrieves the RoleBinding from the db for a given name.
func (p *roleBindingLister) Get(ctx context.Context, name string) (ret *rbac.RoleBinding, err error) {
	a := p.roleBindings
	if i := sort.Search(len(a), func(i int) bool { return a[i].Name >= name }); i < len(a) {
		if ret = a[i]; ret.Name == name {
			return ret, nil
		}
	}

	return nil, errors.NewNotFound("RoleBinding")
}
