package session

import (
	"context"

	"github.com/yubo/apiserver/pkg/models"
	"github.com/yubo/apiserver/pkg/session/types"
	"github.com/yubo/apiserver/pkg/storage"
)

// pkg/registry/rbac/role/storage/storage.go
// pkg/registry/rbac/rest/storage_rbac.go
func NewSession() types.Session {
	o := &session{}
	o.store = models.NewStore(o.Name())
	return o
}

type session struct {
	store models.Store
}

func (p *session) Name() string {
	return "session"
}

func (p *session) NewObj() interface{} {
	return &types.SessionConn{}
}

func (p *session) Create(ctx context.Context, obj *types.SessionConn) (ret *types.SessionConn, err error) {
	err = p.store.Create(ctx, obj.Sid, obj, &ret)
	return
}

// Get retrieves the session from the db for a given sid.
func (p *session) Get(ctx context.Context, sid string) (ret *types.SessionConn, err error) {
	err = p.store.Get(ctx, sid, false, &ret)
	return
}

// List lists all sessions in the indexer.
func (p *session) List(ctx context.Context, opts storage.ListOptions) (list []*types.SessionConn, err error) {
	err = p.store.List(ctx, opts, &list, opts.Total)
	return
}

func (p *session) Update(ctx context.Context, obj *types.SessionConn) (ret *types.SessionConn, err error) {
	err = p.store.Update(ctx, obj.Sid, obj, &ret)
	return
}

func (p *session) Delete(ctx context.Context, sid string) (ret *types.SessionConn, err error) {
	err = p.store.Delete(ctx, sid, &ret)
	return
}

func init() {
	models.Register(&session{})
}
