package listers

import (
	"github.com/yubo/apiserver/pkg/storage"
	"github.com/yubo/apiserver/pkg/api/rbac"
	"github.com/yubo/golib/staging/labels"
	"github.com/yubo/golib/orm"
)

// ClusterRoleBindingLister helps list Roles.
// All objects returned here must be treated as read-only.
type ClusterRoleBindingLister interface {
	// List lists all Roles in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*rbac.ClusterRoleBinding, err error)
	Get(name string) (*rbac.ClusterRoleBinding, error)
}

// clusterRoleBindingLister implements the ClusterRoleBindingLister interface.
type clusterRoleBindingLister struct {
	db *orm.DB
}

// NewClusterRoleBindingLister returns a new ClusterRoleBindingLister.
func NewClusterRoleBindingLister(db *orm.DB) ClusterRoleBindingLister {
	return &clusterRoleBindingLister{db: db}
}

// List lists all Roles in the indexer.
func (s *clusterRoleBindingLister) List(selector labels.Selector) (ret []*rbac.ClusterRoleBinding, err error) {
	err = storage.List(s.db, "cluster_role_binding", selector, &ret)
	return
}

// Get retrieves the ClusterRoleBinding from the db for a given name.
func (s *clusterRoleBindingLister) Get(name string) (ret *rbac.ClusterRoleBinding, err error) {
	err = storage.Get(s.db, "cluster_role_binding", name, &ret)
	return
}
