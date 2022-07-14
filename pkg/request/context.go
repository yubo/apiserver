/*
Copyright 2014 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package request

import (
	"context"
	"net"

	"github.com/emicklei/go-restful/v3"
	"github.com/yubo/apiserver/pkg/apis/audit"
	"github.com/yubo/apiserver/pkg/authentication/user"
	"github.com/yubo/apiserver/pkg/session/types"
	"github.com/yubo/golib/api"
	"go.opentelemetry.io/otel/trace"
	// "github.com/yubo/apiserver/pkg/apis/audit"
)

var (
	noopTracer = trace.NewNoopTracerProvider().Tracer("noop")
)

// The key type is unexported to prevent collisions
type key int

const (
	// namespaceKey is the context key for the request namespace.
	namespaceKey key = iota

	// userKey is the context key for the request user.
	userKey

	// auditKey is the context key for the audit event.
	auditKey

	// resp
	respKey
	tracerKey
	traceIDKey
	sessionKey

	// bodyKey is the context key for the request body
	bodyKey

	// paramKey is the context key for the request param
	paramKey

	// connection
	connKey
)

// NewContext instantiates a base context object for request flows.
func NewContext() context.Context {
	return context.TODO()
}

// NewDefaultContext instantiates a base context object for request flows in the default namespace
func NewDefaultContext() context.Context {
	return WithNamespace(NewContext(), api.NamespaceDefault)
}

// WithValue returns a copy of parent in which the value associated with key is val.
func WithValue(parent context.Context, key interface{}, val interface{}) context.Context {
	return context.WithValue(parent, key, val)
}

// WithNamespace returns a copy of parent in which the namespace value is set
func WithNamespace(parent context.Context, namespace string) context.Context {
	return WithValue(parent, namespaceKey, namespace)
}

// NamespaceFrom returns the value of the namespace key on the ctx
func NamespaceFrom(ctx context.Context) (string, bool) {
	namespace, ok := ctx.Value(namespaceKey).(string)
	return namespace, ok
}

// NamespaceValue returns the value of the namespace key on the ctx, or the empty string if none
func NamespaceValue(ctx context.Context) string {
	namespace, _ := NamespaceFrom(ctx)
	return namespace
}

// WithUser returns a copy of parent in which the user value is set
func WithUser(parent context.Context, user user.Info) context.Context {
	return WithValue(parent, userKey, user)
}

// UserFrom returns the value of the user key on the ctx
func UserFrom(ctx context.Context) (user.Info, bool) {
	user, ok := ctx.Value(userKey).(user.Info)
	return user, ok
}

// WithAuditEvent returns set audit event struct.
func WithAuditEvent(parent context.Context, ev *audit.Event) context.Context {
	return WithValue(parent, auditKey, ev)
}

// AuditEventFrom returns the audit event struct on the ctx
func AuditEventFrom(ctx context.Context) *audit.Event {
	ev, _ := ctx.Value(auditKey).(*audit.Event)
	return ev
}

// WithResp returns a copy of parent in which the response value is set
func WithResp(parent context.Context, resp *restful.Response) context.Context {
	return WithValue(parent, respKey, resp)
}

// RespFrom returns the value of the response key on the ctx
func RespFrom(ctx context.Context) (*restful.Response, bool) {
	resp, ok := ctx.Value(respKey).(*restful.Response)
	return resp, ok
}

// WithTracer returns a copy of parent in which the tracer value is set
func WithTracer(parent context.Context, tracer trace.Tracer) context.Context {
	return context.WithValue(parent, tracerKey, tracer)
}

// TracerFrom returns the value of the tracer key on the ctx
func TracerFrom(ctx context.Context) trace.Tracer {
	if tracer, ok := ctx.Value(tracerKey).(trace.Tracer); ok {
		return tracer
	}
	return noopTracer
}

// WithTraceID returns a copy of parent in which the traceID value is set
func WithTraceID(parent context.Context, traceID string) context.Context {
	return context.WithValue(parent, traceIDKey, traceID)
}

// TraceIDFrom returns the value of the traceID key on the ctx
func TraceIDFrom(ctx context.Context) (string, bool) {
	traceID, ok := ctx.Value(traceIDKey).(string)
	return traceID, ok
}

// WithrSession returns a copy of parent in which the session value is set
func WithSession(parent context.Context, sess types.SessionContext) context.Context {
	return WithValue(parent, sessionKey, sess)
}

// SessionFrom returns the value of the session key on the ctx
func SessionFrom(ctx context.Context) (types.SessionContext, bool) {
	s, ok := ctx.Value(sessionKey).(types.SessionContext)
	return s, ok
}

// WithBody returns a copy of parent in which the body value is set
func WithBody(parent context.Context, body interface{}) context.Context {
	return WithValue(parent, bodyKey, body)
}

// BodyFrom returns the value of the param key on the ctx
func BodyFrom(ctx context.Context) interface{} {
	return ctx.Value(bodyKey)
}

// WithParam returns a copy of parent in which the param value is set
func WithParam(parent context.Context, param interface{}) context.Context {
	return WithValue(parent, paramKey, param)
}

// ParamFrom returns the value of the param key on the ctx
func ParamFrom(ctx context.Context) interface{} {
	return ctx.Value(paramKey)
}

// WithConn returns a copy of parent in which the param value is set
func WithConn(parent context.Context, conn net.Conn) context.Context {
	return WithValue(parent, connKey, conn)
}

// ConnFrom returns the value of the param key on the ctx
func ConnFrom(ctx context.Context) (net.Conn, bool) {
	conn, ok := ctx.Value(connKey).(net.Conn)
	return conn, ok
}
