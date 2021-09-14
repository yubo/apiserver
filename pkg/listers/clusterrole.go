package listers

import (
	"github.com/yubo/apiserver/pkg/api/rbac"
	"github.com/yubo/golib/labels"
)

// ClusterRoleLister helps list Roles.
// All objects returned here must be treated as read-only.
type ClusterRoleLister interface {
	// List lists all Roles in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*rbac.ClusterRole, err error)
	Get(name string) (*rbac.ClusterRole, error)
}
