package options

import (
	"context"

	// authentication "github.com/yubo/apiserver/modules/authentication/lib"

	"github.com/yubo/apiserver/pkg/audit"
	"github.com/yubo/apiserver/pkg/db"
	"github.com/yubo/apiserver/pkg/dynamiccertificates"
	"github.com/yubo/apiserver/pkg/server"
	"github.com/yubo/apiserver/pkg/session"
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
	clientCAKey       // clientCA
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
func WithAuthn(ctx context.Context, authn *server.AuthenticationInfo) {
	klog.V(5).Infof("attr with authn")
	proc.AttrMustFrom(ctx)[authnKey] = authn
}

// AuthnFrom returns the value of the authenticationInfo key on the ctx
func AuthnFrom(ctx context.Context) (*server.AuthenticationInfo, bool) {
	authn, ok := proc.AttrMustFrom(ctx)[authnKey].(*server.AuthenticationInfo)
	return authn, ok
}

// WithAuthz returns a copy of ctx in which the authorizationInfo value is set
func WithAuthz(ctx context.Context, authz *server.AuthorizationInfo) {
	klog.V(5).Infof("attr with authz")
	proc.AttrMustFrom(ctx)[authzKey] = authz
}

// AuthzFrom returns the value of the authorizationInfo key on the ctx
func AuthzFrom(ctx context.Context) (*server.AuthorizationInfo, bool) {
	authz, ok := proc.AttrMustFrom(ctx)[authzKey].(*server.AuthorizationInfo)
	return authz, ok
}

// WithAudit returns a copy of ctx in which the audit value is set
func WithAudit(ctx context.Context, audit audit.Auditor) {
	klog.V(5).Infof("attr with audit")
	proc.AttrMustFrom(ctx)[auditKey] = audit
}

// AuditFrom returns the value of the audit key on the ctx
func AuditFrom(ctx context.Context) (audit.Auditor, bool) {
	authz, ok := proc.AttrMustFrom(ctx)[auditKey].(audit.Auditor)
	return authz, ok
}

// WithAPIServer returns a copy of ctx in which the http value is set
func WithAPIServer(ctx context.Context, server server.APIServer) {
	klog.V(5).Infof("attr with server")
	proc.AttrMustFrom(ctx)[apiServerKey] = server
}

// APIServerFrom returns the value of the http key on the ctx
func APIServerFrom(ctx context.Context) (server.APIServer, bool) {
	server, ok := proc.AttrMustFrom(ctx)[apiServerKey].(server.APIServer)
	return server, ok
}

func APIServerMustFrom(ctx context.Context) server.APIServer {
	server, ok := proc.AttrMustFrom(ctx)[apiServerKey].(server.APIServer)
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

func WithDB(ctx context.Context, db db.DB) {
	klog.V(5).Infof("attr with db %v attr %p", db, proc.AttrMustFrom(ctx))
	proc.AttrMustFrom(ctx)[dbKey] = db
}

func DBFrom(ctx context.Context, name string) (db.DB, bool) {
	klog.V(5).Infof("attr with name %v attr %p", name, proc.AttrMustFrom(ctx))
	d, ok := proc.AttrMustFrom(ctx)[dbKey].(db.DB)
	if !ok {
		return nil, false
	}
	if name == "" {
		name = db.DefaultName
	}

	if d = d.GetDB(name); d == nil {
		return nil, false
	}
	return d, true
}

func DBMustFrom(ctx context.Context, name string) db.DB {
	db, ok := DBFrom(ctx, name)
	if !ok {
		panic("unable get db." + name)
	}
	return db
}

func WithClientCA(ctx context.Context, clientCA dynamiccertificates.CAContentProvider) {
	klog.V(5).Infof("attr with clientCA")
	proc.AttrMustFrom(ctx)[clientCAKey] = clientCA
}

func ClientCAFrom(ctx context.Context) (dynamiccertificates.CAContentProvider, bool) {
	clientCA, ok := proc.AttrMustFrom(ctx)[clientCAKey].(dynamiccertificates.CAContentProvider)
	return clientCA, ok
}
