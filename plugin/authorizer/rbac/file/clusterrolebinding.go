package file

import (
	"context"
	"sort"

	"github.com/yubo/apiserver/pkg/apis/rbac"
	"github.com/yubo/apiserver/pkg/listers"
	"github.com/yubo/golib/api"
	"github.com/yubo/golib/api/errors"
)

// clusterRoleBindingLister implements the ClusterRoleBindingLister interface.
type clusterRoleBindingLister struct {
	*FileStorage
}

// NewClusterRoleBindingLister returns a new ClusterRoleBindingLister.
func NewClusterRoleBindingLister(f *FileStorage) listers.ClusterRoleBindingLister {
	return &clusterRoleBindingLister{FileStorage: f}
}

// List lists all ClusterRoleBinding in the indexer.
func (p *clusterRoleBindingLister) List(ctx context.Context, opts api.GetListOptions) (list []*rbac.ClusterRoleBinding, err error) {
	return p.clusterRoleBindings, nil
}

// Get retrieves the ClusterRoleBinding from the db for a given name.
func (p *clusterRoleBindingLister) Get(ctx context.Context, name string) (ret *rbac.ClusterRoleBinding, err error) {
	a := p.clusterRoleBindings
	if i := sort.Search(len(a), func(i int) bool { return a[i].Name >= name }); i < len(a) {
		if ret = a[i]; ret.Name == name {
			return ret, nil
		}
	}

	return nil, errors.NewNotFound("ClusterRoleBinding")
}
