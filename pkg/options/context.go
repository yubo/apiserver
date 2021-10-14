package options

import (
	"context"

	// authentication "github.com/yubo/apiserver/modules/authentication/lib"

	"github.com/yubo/apiserver/pkg/authorization/rbac"
	"github.com/yubo/golib/net/session"
	"github.com/yubo/golib/orm"
	"github.com/yubo/golib/proc"
	"google.golang.org/grpc"
	"k8s.io/klog/v2"
)

// The key type is unexported to prevent collisions
type key int

const (
	_ key = iota
	dbKey
	apiServerKey
	grpcserverKey
	authnKey          // Authentication
	sessionManagerKey //
	authzKey          // authorization
	auditKey          // audit
	rbacKey           // authz-rbac
)

// WithValue returns a copy of parent in which the value associated with key is val.
func WithValue(parent context.Context, key interface{}, val interface{}) context.Context {
	return context.WithValue(parent, key, val)
}

// WithGrpcServer returns a copy of ctx in which the grpc value is set
func WithGrpcServer(ctx context.Context, g *grpc.Server) {
	klog.V(5).Infof("attr with grpc")
	proc.AttrMustFrom(ctx)[grpcserverKey] = g
}

// GrpcServerFrom returns the value of the grpc key on the ctx
func GrpcServerFrom(ctx context.Context) (*grpc.Server, bool) {
	g, ok := proc.AttrMustFrom(ctx)[grpcserverKey].(*grpc.Server)
	return g, ok
}

// WithAuthn returns a copy of ctx in which the authenticationInfo value is set
func WithAuthn(ctx context.Context, authn Authn) {
	klog.V(5).Infof("attr with authn")
	proc.AttrMustFrom(ctx)[authnKey] = authn
}

// AuthnFrom returns the value of the authenticationInfo key on the ctx
func AuthnFrom(ctx context.Context) (Authn, bool) {
	authn, ok := proc.AttrMustFrom(ctx)[authnKey].(Authn)
	return authn, ok
}

// WithAuthz returns a copy of ctx in which the authorizationInfo value is set
func WithAuthz(ctx context.Context, authz Authz) {
	klog.V(5).Infof("attr with authz")
	proc.AttrMustFrom(ctx)[authzKey] = authz
}

// AuthzFrom returns the value of the authorizationInfo key on the ctx
func AuthzFrom(ctx context.Context) (Authz, bool) {
	authz, ok := proc.AttrMustFrom(ctx)[authzKey].(Authz)
	return authz, ok
}

// WithAudit returns a copy of ctx in which the audit value is set
func WithAudit(ctx context.Context, audit Audit) {
	klog.V(5).Infof("attr with audit")
	proc.AttrMustFrom(ctx)[auditKey] = audit
}

// AuditFrom returns the value of the audit key on the ctx
func AuditFrom(ctx context.Context) (Audit, bool) {
	authz, ok := proc.AttrMustFrom(ctx)[auditKey].(Audit)
	return authz, ok
}

// WithApiServer returns a copy of ctx in which the http value is set
func WithApiServer(ctx context.Context, server ApiServer) {
	klog.V(5).Infof("attr with apiserver")
	proc.AttrMustFrom(ctx)[apiServerKey] = server
}

// ApiServerFrom returns the value of the http key on the ctx
func ApiServerFrom(ctx context.Context) (ApiServer, bool) {
	server, ok := proc.AttrMustFrom(ctx)[apiServerKey].(ApiServer)
	return server, ok
}

func ApiServerMustFrom(ctx context.Context) ApiServer {
	server, ok := proc.AttrMustFrom(ctx)[apiServerKey].(ApiServer)
	if !ok {
		panic("unable get genericServer")
	}
	return server
}

func WithSessionManager(ctx context.Context, sm session.SessionManager) {
	klog.V(5).Infof("attr with session manager")
	proc.AttrMustFrom(ctx)[sessionManagerKey] = sm
}

func SessionManagerFrom(ctx context.Context) (session.SessionManager, bool) {
	sm, ok := proc.AttrMustFrom(ctx)[sessionManagerKey].(session.SessionManager)
	return sm, ok
}

func WithDB(ctx context.Context, db orm.DB) {
	klog.V(5).Infof("attr with db")
	proc.AttrMustFrom(ctx)[dbKey] = db
}

func DBFrom(ctx context.Context) (orm.DB, bool) {
	db, ok := proc.AttrMustFrom(ctx)[dbKey].(orm.DB)
	return db, ok
}

func DBMustFrom(ctx context.Context) orm.DB {
	db, ok := proc.AttrMustFrom(ctx)[dbKey].(orm.DB)
	if !ok {
		panic("unable get db")
	}
	return db
}

func WithRBAC(ctx context.Context, r *rbac.RBACAuthorizer) {
	klog.V(5).Infof("attr with rbac")
	proc.AttrMustFrom(ctx)[rbacKey] = r
}

func RBACFrom(ctx context.Context) (*rbac.RBACAuthorizer, bool) {
	v, ok := proc.AttrMustFrom(ctx)[rbacKey].(*rbac.RBACAuthorizer)
	return v, ok
}

func RBACMustFrom(ctx context.Context) *rbac.RBACAuthorizer {
	v, ok := proc.AttrMustFrom(ctx)[rbacKey].(*rbac.RBACAuthorizer)
	if !ok {
		panic("unable get RBACAuthorizer")
	}
	return v
}
