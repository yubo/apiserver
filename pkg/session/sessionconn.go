package session

import (
	"context"

	"github.com/yubo/apiserver/pkg/models"
	"github.com/yubo/apiserver/pkg/session/types"
	"github.com/yubo/apiserver/pkg/storage"
	"github.com/yubo/golib/orm"
)

// pkg/registry/rbac/role/storage/storage.go
// pkg/registry/rbac/rest/storage_rbac.go
func NewSessionConn() *SessionConn {
	return &SessionConn{DB: models.DB()}
}

type SessionConn struct {
	orm.DB
}

func (p *SessionConn) Name() string {
	return "session_conn"
}

func (p *SessionConn) NewObj() interface{} {
	return &types.SessionConn{}
}

func (p *SessionConn) Create(ctx context.Context, obj *types.SessionConn) error {
	return p.Insert(ctx, obj, orm.WithTable(p.Name()))
}

// Get retrieves the session from the db for a given sid.
func (p *SessionConn) Get(ctx context.Context, sid string) (ret *types.SessionConn, err error) {
	err = p.Query(ctx, "select * from session_conn where sid=?", sid).Row(&ret)
	return
}

// List lists all sessions in the indexer.
func (p *SessionConn) List(ctx context.Context, o *storage.ListOptions, opts ...orm.Option) (list []types.SessionConn, err error) {
	err = p.DB.List(ctx, &list, append(opts,
		orm.WithTable(p.Name()),
		orm.WithTotal(o.Total),
		orm.WithSelector(o.Query),
		orm.WithOrderby(o.Orderby...),
		orm.WithLimit(o.Offset, o.Limit))...,
	)
	return
}

func (p *SessionConn) Update(ctx context.Context, obj *types.SessionConn) error {
	return p.DB.Update(ctx, obj, orm.WithTable(p.Name()))
}

func (p *SessionConn) Delete(ctx context.Context, sid string) error {
	_, err := p.Exec(ctx, "delete from session_conn where sid=?", sid)
	return err
}

func (p *SessionConn) Clean(ctx context.Context, expiresAt int64, cookieName string) error {
	_, err := p.Exec(ctx, "delete from session_conn where updated_at<? and cookie_name=?", expiresAt, cookieName)
	return err
}
