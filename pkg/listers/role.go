package listers

import (
	"github.com/yubo/apiserver/pkg/apis/rbac"
	"github.com/yubo/golib/labels"
)

// RoleLister helps list Roles.
// All objects returned here must be treated as read-only.
type RoleLister interface {
	// List lists all Roles in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*rbac.Role, err error)
	Get(name string) (*rbac.Role, error)
}
