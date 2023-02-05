package listers

import (
	"context"

	"github.com/yubo/apiserver/pkg/apis/rbac"
	"github.com/yubo/golib/api"
)

// ClusterRoleLister helps list Roles.
// All objects returned here must be treated as read-only.
type ClusterRoleLister interface {
	// List lists all Roles in the indexer.
	// Objects returned here must be treated as read-only.
	List(ctx context.Context, opts api.GetListOptions) (list []*rbac.ClusterRole, err error)
	Get(ctx context.Context, name string) (*rbac.ClusterRole, error)
}
