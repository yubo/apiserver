package listers

import (
	"context"

	"github.com/yubo/apiserver/pkg/apis/rbac"
	"github.com/yubo/golib/api"
)

// RoleBindingLister helps list Roles.
// All objects returned here must be treated as read-only.
type RoleBindingLister interface {
	// List lists all Roles in the indexer.
	// Objects returned here must be treated as read-only.
	List(ctx context.Context, opts api.GetListOptions) (list []*rbac.RoleBinding, err error)
	Get(ctx context.Context, name string) (*rbac.RoleBinding, error)
}
