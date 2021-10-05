package db

import (
	"github.com/yubo/apiserver/pkg/api/rbac"
	"github.com/yubo/apiserver/pkg/listers"
	"github.com/yubo/apiserver/pkg/storage"
	"github.com/yubo/golib/labels"
	"github.com/yubo/golib/orm"
)

// clusterRoleLister implements the ClusterRoleLister interface.
type clusterRoleLister struct {
	db orm.DB
}

// NewClusterRoleLister returns a new ClusterRoleLister.
func NewClusterRoleLister(db orm.DB) listers.ClusterRoleLister {
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
