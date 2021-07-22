package listers

import (
	"github.com/yubo/apiserver/pkg/storage"
	"github.com/yubo/apiserver/pkg/api/rbac"
	"github.com/yubo/golib/staging/labels"
	"github.com/yubo/golib/orm"
)

// ClusterRoleLister helps list Roles.
// All objects returned here must be treated as read-only.
type ClusterRoleLister interface {
	// List lists all Roles in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*rbac.ClusterRole, err error)
	Get(name string) (*rbac.ClusterRole, error)
}

// clusterRoleLister implements the ClusterRoleLister interface.
type clusterRoleLister struct {
	db *orm.DB
}

// NewClusterRoleLister returns a new ClusterRoleLister.
func NewClusterRoleLister(db *orm.DB) ClusterRoleLister {
	return &clusterRoleLister{db: db}
}

// List lists all Roles in the indexer.
func (s *clusterRoleLister) List(selector labels.Selector) (ret []*rbac.ClusterRole, err error) {
	err = storage.List(s.db, "cluster_role", selector, &ret)
	return
}

// Get retrieves the ClusterRole from the db for a given name.
func (s *clusterRoleLister) Get(name string) (ret *rbac.ClusterRole, err error) {
	err = storage.Get(s.db, "cluster_role", name, &ret)
	return
}
