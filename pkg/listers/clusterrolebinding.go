package listers

import (
	"context"

	"github.com/yubo/apiserver/pkg/apis/rbac"
	"github.com/yubo/apiserver/pkg/storage"
)

// ClusterRoleBindingLister helps list Roles.
// All objects returned here must be treated as read-only.
type ClusterRoleBindingLister interface {
	// List lists all Roles in the indexer.
	// Objects returned here must be treated as read-only.
	List(ctx context.Context, opts storage.ListOptions) (total int64, list []*rbac.ClusterRoleBinding, err error)
	Get(ctx context.Context, name string) (*rbac.ClusterRoleBinding, error)
}
