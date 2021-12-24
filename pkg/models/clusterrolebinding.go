package models

import (
	"github.com/yubo/apiserver/pkg/apis/rbac"
	"github.com/yubo/apiserver/pkg/listers"
	"github.com/yubo/apiserver/pkg/storage"
	"github.com/yubo/golib/labels"
	"github.com/yubo/golib/orm"
)

// clusterRoleBindingLister implements the ClusterRoleBindingLister interface.
type clusterRoleBindingLister struct {
	db orm.DB
}

// NewClusterRoleBindingLister returns a new ClusterRoleBindingLister.
func NewClusterRoleBindingLister(db orm.DB) listers.ClusterRoleBindingLister {
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
