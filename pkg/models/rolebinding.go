package models

import (
	"github.com/yubo/apiserver/pkg/apis/rbac"
	"github.com/yubo/apiserver/pkg/listers"
	"github.com/yubo/apiserver/pkg/storage"
	"github.com/yubo/golib/labels"
)

// roleBinding implements the RoleBinding interface.
type roleBinding struct{}

// NewRoleBindingLister returns a new RoleBindingLister.
func NewRoleBindingLister() listers.RoleBindingLister {
	return &roleBinding{}
}

// List lists all Roles in the indexer.
func (s *roleBinding) List(selector labels.Selector) (ret []*rbac.RoleBinding, err error) {
	err = storage.List(s.db, "cluster_role", selector, &ret)
	return
}

// Get retrieves the RoleBinding from the db for a given name.
func (s *roleBinding) Get(name string) (ret *rbac.RoleBinding, err error) {
	err = storage.Get(s.db, "cluster_role", name, &ret)
	return
}
