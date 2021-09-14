package db

import (
	"github.com/yubo/apiserver/pkg/authorization/rbac"
	"github.com/yubo/golib/orm"
)

type Config struct {
	ConfigPath string `json:"configPath"`
}

type FileStorage struct {
	*Config
}

func NewRBAC(db *orm.DB) (*rbac.RBACAuthorizer, error) {

	return rbac.New(
		&rbac.RoleGetter{Lister: NewRoleLister(db)},
		&rbac.RoleBindingLister{Lister: NewRoleBindingLister(db)},
		&rbac.ClusterRoleGetter{Lister: NewClusterRoleLister(db)},
		&rbac.ClusterRoleBindingLister{Lister: NewClusterRoleBindingLister(db)},
	), nil
}

func NewFileStorage(config *Config) (*FileStorage, error) {
	return &FileStorage{Config: config}, nil
}
