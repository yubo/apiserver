package types

import (
	"context"
	"net/http"
	"time"

	"github.com/yubo/apiserver/pkg/storage"
)

type SessionConn struct {
	Sid        string `sql:"name,where,primary_key"`
	UserName   string
	Data       map[string]string
	CookieName string
	CreatedAt  int64
	UpdatedAt  int64
}

type SessionContext interface {
	Set(key, value string) error
	Get(key string) string
	CreatedAt() time.Time
	Delete(key string) error
	Reset() error
	Sid() string
	Update(w http.ResponseWriter) error
}

// session model
type Session interface {
	Name() string
	NewObj() interface{}

	List(ctx context.Context, opts storage.ListOptions) ([]*SessionConn, error)
	Get(ctx context.Context, sid string) (*SessionConn, error)
	Create(ctx context.Context, obj *SessionConn) (*SessionConn, error)
	Update(ctx context.Context, obj *SessionConn) (*SessionConn, error)
	Delete(ctx context.Context, sid string) (*SessionConn, error)
}

type SessionManager interface {
	// start a session connection
	Start(w http.ResponseWriter, r *http.Request) (SessionContext, error)
	// stop session manager
	Stop()
	// start session connection GC
	GC()
	Destroy(w http.ResponseWriter, r *http.Request) error
	Get(ctx context.Context, sid string) (SessionContext, error)
}
