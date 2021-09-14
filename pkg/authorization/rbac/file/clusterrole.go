package file

import (
	"sort"

	"github.com/yubo/apiserver/pkg/api/rbac"
	"github.com/yubo/apiserver/pkg/listers"
	"github.com/yubo/golib/api/errors"
	"github.com/yubo/golib/labels"
)

// clusterRoleLister implements the ClusterRoleLister interface.
type clusterRoleLister struct {
	*FileStorage
}

// NewClusterRoleLister returns a new ClusterRoleLister.
func NewClusterRoleLister(f *FileStorage) listers.ClusterRoleLister {
	return &clusterRoleLister{FileStorage: f}
}

// List lists all ClusterRoles in the indexer.
func (p *clusterRoleLister) List(selector labels.Selector) (ret []*rbac.ClusterRole, err error) {
	return p.clusterRoles, nil
}

// Get retrieves the ClusterRole from the db for a given name.
func (p *clusterRoleLister) Get(name string) (ret *rbac.ClusterRole, err error) {
	a := p.clusterRoles
	i := sort.Search(len(a), func(i int) bool { return a[i].Name >= name })
	if ret = a[i]; ret.Name == name {
		return ret, nil
	}

	return nil, errors.NewNotFound("ClusterRole")
}
