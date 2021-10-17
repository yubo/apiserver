package listers

import (
	"github.com/yubo/apiserver/pkg/apis/rbac"
	"github.com/yubo/golib/labels"
)

// ClusterRoleBindingLister helps list Roles.
// All objects returned here must be treated as read-only.
type ClusterRoleBindingLister interface {
	// List lists all Roles in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*rbac.ClusterRoleBinding, err error)
	Get(name string) (*rbac.ClusterRoleBinding, error)
}
