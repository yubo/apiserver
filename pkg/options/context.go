package options

import (
	"context"

	// authentication "github.com/yubo/apiserver/modules/authentication/lib"

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
)

// WithValue returns a copy of parent in which the value associated with key is val.
func WithValue(parent context.Context, key interface{}, val interface{}) context.Context {
	return context.WithValue(parent, key, val)
}

// WithGrpcServer returns a copy of ctx in which the grpc value is set
func WithGrpcServer(ctx context.Context, g *grpc.Server) {
	klog.V(5).Infof("attr with grpc")
	proc.AttrFrom(ctx)[grpcserverKey] = g
}

// GrpcServerFrom returns the value of the grpc key on the ctx
func GrpcServerFrom(ctx context.Context) (*grpc.Server, bool) {
	g, ok := proc.AttrFrom(ctx)[grpcserverKey].(*grpc.Server)
	return g, ok
}

// WithAuthn returns a copy of ctx in which the authenticationInfo value is set
func WithAuthn(ctx context.Context, authn Authn) {
	klog.V(5).Infof("attr with authn")
	proc.AttrFrom(ctx)[authnKey] = authn
}

// AuthnFrom returns the value of the authenticationInfo key on the ctx
func AuthnFrom(ctx context.Context) (Authn, bool) {
	authn, ok := proc.AttrFrom(ctx)[authnKey].(Authn)
	return authn, ok
}

// WithAuthz returns a copy of ctx in which the authorizationInfo value is set
func WithAuthz(ctx context.Context, authz Authz) {
	klog.V(5).Infof("attr with authz")
	proc.AttrFrom(ctx)[authzKey] = authz
}

// AuthzFrom returns the value of the authorizationInfo key on the ctx
func AuthzFrom(ctx context.Context) (Authz, bool) {
	authz, ok := proc.AttrFrom(ctx)[authzKey].(Authz)
	return authz, ok
}

// WithApiServer returns a copy of ctx in which the http value is set
func WithApiServer(ctx context.Context, server ApiServer) {
	klog.V(5).Infof("attr with apiserver")
	proc.AttrFrom(ctx)[apiServerKey] = server
}

// ApiServerFrom returns the value of the http key on the ctx
func ApiServerFrom(ctx context.Context) (ApiServer, bool) {
	server, ok := proc.AttrFrom(ctx)[apiServerKey].(ApiServer)
	return server, ok
}

func ApiServerMustFrom(ctx context.Context) ApiServer {
	server, ok := proc.AttrFrom(ctx)[apiServerKey].(ApiServer)
	if !ok {
		panic("unable get genericServer")
	}
	return server
}

func WithSessionManager(ctx context.Context, sm session.SessionManager) {
	klog.V(5).Infof("attr with session manager")
	proc.AttrFrom(ctx)[sessionManagerKey] = sm
}

func SessionManagerFrom(ctx context.Context) (session.SessionManager, bool) {
	sm, ok := proc.AttrFrom(ctx)[sessionManagerKey].(session.SessionManager)
	return sm, ok
}

func WithDB(ctx context.Context, db *orm.DB) {
	klog.V(5).Infof("attr with db")
	proc.AttrFrom(ctx)[dbKey] = db
}

func DBFrom(ctx context.Context) (*orm.DB, bool) {
	db, ok := proc.AttrFrom(ctx)[dbKey].(*orm.DB)
	return db, ok
}

func DBMustFrom(ctx context.Context) *orm.DB {
	db, ok := proc.AttrFrom(ctx)[dbKey].(*orm.DB)
	if !ok {
		panic("unable get db")
	}
	return db
}
