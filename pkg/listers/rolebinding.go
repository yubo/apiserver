package listers

import (
	"github.com/yubo/apiserver/pkg/api/rbac"
	"github.com/yubo/golib/labels"
)

// RoleBindingLister helps list Roles.
// All objects returned here must be treated as read-only.
type RoleBindingLister interface {
	// List lists all Roles in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*rbac.RoleBinding, err error)
	Get(name string) (*rbac.RoleBinding, error)
}
