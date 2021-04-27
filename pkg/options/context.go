package options

import (
	"context"

	// authentication "github.com/yubo/apiserver/modules/authentication/lib"

	"github.com/yubo/golib/net/session"
	"github.com/yubo/golib/orm"
	"google.golang.org/grpc"
)

// The key type is unexported to prevent collisions
type key int

const (
	_ key = iota
	secureKey
	httpServerKey
	genericServerKey
	dbKey
	grpcServerKey
	respKey
	authenticationKey
	sessionManagerKey
	authorizationKey
	//tracerKey
)

// WithValue returns a copy of parent in which the value associated with key is val.
func WithValue(parent context.Context, key interface{}, val interface{}) context.Context {
	return context.WithValue(parent, key, val)
}

// WithSecureServing returns a copy of parent in which the secure value is set
func WithSecureServing(parent context.Context, secure SecureServing) context.Context {
	return WithValue(parent, secureKey, secure)
}

// SecureServingFrom returns the value of the secure key on the ctx
func SecureServingFrom(ctx context.Context) (SecureServing, bool) {
	secure, ok := ctx.Value(secureKey).(SecureServing)
	return secure, ok
}

// WithHttpServer returns a copy of parent in which the http value is set
func WithHttpServer(parent context.Context, http HttpServer) context.Context {
	return WithValue(parent, httpServerKey, http)
}

// HttpServerFrom returns the value of the http key on the ctx
func HttpServerFrom(ctx context.Context) (HttpServer, bool) {
	http, ok := ctx.Value(httpServerKey).(HttpServer)
	return http, ok
}

func HttpServerMustFrom(ctx context.Context) HttpServer {
	http, ok := ctx.Value(httpServerKey).(HttpServer)
	if !ok {
		panic("unable get httpServer")
	}
	return http
}

// WithDB returns a copy of parent in which the db value is set
func WithDB(parent context.Context, db *orm.DB) context.Context {
	return WithValue(parent, dbKey, db)
}

// DBFrom returns the value of the db key on the ctx
func DBFrom(ctx context.Context) (*orm.DB, bool) {
	db, ok := ctx.Value(dbKey).(*orm.DB)
	return db, ok
}

func DBMustFrom(ctx context.Context) *orm.DB {
	db, ok := ctx.Value(dbKey).(*orm.DB)
	if !ok {
		panic("unable get db")
	}
	return db
}

// WithGrpcServer returns a copy of parent in which the grpc value is set
func WithGrpcServer(parent context.Context, g *grpc.Server) context.Context {
	return WithValue(parent, grpcServerKey, g)
}

// GrpcServerFrom returns the value of the grpc key on the ctx
func GrpcServerFrom(ctx context.Context) (*grpc.Server, bool) {
	g, ok := ctx.Value(grpcServerKey).(*grpc.Server)
	return g, ok
}

// WithAuthn returns a copy of parent in which the authenticationInfo value is set
func WithAuthn(parent context.Context, authn Authn) context.Context {
	return WithValue(parent, authenticationKey, authn)
}

// AuthnFrom returns the value of the authenticationInfo key on the ctx
func AuthnFrom(ctx context.Context) (Authn, bool) {
	authn, ok := ctx.Value(authenticationKey).(Authn)
	return authn, ok
}

// WithAuthz returns a copy of parent in which the authorizationInfo value is set
func WithAuthz(parent context.Context, authz Authz) context.Context {
	return WithValue(parent, authorizationKey, authz)
}

// AuthzFrom returns the value of the authorizationInfo key on the ctx
func AuthzFrom(ctx context.Context) (Authz, bool) {
	authz, ok := ctx.Value(authorizationKey).(Authz)
	return authz, ok
}

// WithGenericServer returns a copy of parent in which the http value is set
func WithGenericServer(parent context.Context, http GenericServer) context.Context {
	return WithValue(parent, genericServerKey, http)
}

// GenericServerFrom returns the value of the http key on the ctx
func GenericServerFrom(ctx context.Context) (GenericServer, bool) {
	http, ok := ctx.Value(genericServerKey).(GenericServer)
	return http, ok
}

func GenericServerMustFrom(ctx context.Context) GenericServer {
	http, ok := ctx.Value(genericServerKey).(GenericServer)
	if !ok {
		panic("unable get genericServer")
	}
	return http
}

// WithTracer returns a copy of parent in which the tracer value is set
//func WithTracer(parent context.Context, tracer opentracing.Tracer) context.Context {
//	return WithValue(parent, tracerKey, tracer)
//}
//
//// TracerFrom returns the value of the tracer key on the ctx
//func TracerFrom(ctx context.Context) (opentracing.Tracer, bool) {
//	tracer, ok := ctx.Value(tracerKey).(opentracing.Tracer)
//	return tracer, ok
//}

// WithrSessionManager returns a copy of parent in which the sessionManager value is set
func WithSessionManager(parent context.Context, manager session.SessionManager) context.Context {
	return WithValue(parent, sessionManagerKey, manager)
}

// SessionManagerFrom returns the value of the sessionManager key on the ctx
func SessionManagerFrom(ctx context.Context) (session.SessionManager, bool) {
	manager, ok := ctx.Value(sessionManagerKey).(session.SessionManager)
	return manager, ok
}
