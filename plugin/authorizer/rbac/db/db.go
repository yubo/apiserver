package db

import (
	"github.com/yubo/apiserver/pkg/models"
	"github.com/yubo/apiserver/plugin/authorizer/rbac"
)

func NewRBAC() (*rbac.RBACAuthorizer, error) {
	return rbac.New(
		&rbac.RoleGetter{Lister: models.NewRole()},
		&rbac.RoleBindingLister{Lister: models.NewRoleBinding()},
		&rbac.ClusterRoleGetter{Lister: models.NewClusterRole()},
		&rbac.ClusterRoleBindingLister{Lister: models.NewClusterRoleBinding()},
	), nil
}
