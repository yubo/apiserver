package file

import (
	"context"
	"sort"

	"github.com/yubo/apiserver/pkg/apis/rbac"
	"github.com/yubo/apiserver/pkg/listers"
	"github.com/yubo/golib/api"
	"github.com/yubo/golib/api/errors"
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
func (p *clusterRoleLister) List(ctx context.Context, opts api.GetListOptions) ([]*rbac.ClusterRole, error) {
	return p.clusterRoles, nil
}

// Get retrieves the ClusterRole from the db for a given name.
func (p *clusterRoleLister) Get(ctx context.Context, name string) (ret *rbac.ClusterRole, err error) {
	a := p.clusterRoles
	if i := sort.Search(len(a), func(i int) bool { return a[i].Name >= name }); i < len(a) {
		if ret = a[i]; ret.Name == name {
			return ret, nil
		}
	}

	return nil, errors.NewNotFound("ClusterRole")
}
