package storage

// k8s.io/apiserver/pkg/registry/generic/registry/store.go
// k8s.io/apiserver/pkg/storage/interfaces.go
// k8s.io/kubernetespkg/registry/core/node/storage/storage.go
import (
	"context"

	"github.com/yubo/golib/api"
	"github.com/yubo/golib/runtime"
)

// Store offers a common interface for object marshaling/unmarshaling operations and
// hides all the storage-related operations behind it.
type Store interface {
	Create(ctx context.Context, key string, obj, out runtime.Object) error

	Delete(ctx context.Context, key string, out runtime.Object) error

	Update(ctx context.Context, key string, obj, out runtime.Object) error

	Get(ctx context.Context, key string, opts api.GetOptions, out runtime.Object) error

	List(ctx context.Context, key string, opts api.GetListOptions, out runtime.Object, total *int) error
}

// GetOptions provides the options that may be provided for storage get operations.
//type GetOptions struct {
//	// IgnoreNotFound determines what is returned if the requested object is not found. If
//	// true, a zero object is returned. If false, an error is returned.
//	IgnoreNotFound bool
//}

// ListOptions provides the options that may be provided for storage list operations.
//type ListOptions struct {
//	Query   string
//	Orderby []string
//	Offset  int
//	Limit   int
//
//	// for output count(*)
//	Total *int
//}
