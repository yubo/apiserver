// this is a sample echo rest api module
//    /api/v1/users --> getUser1
//    /api/v2/users --> getUser2 --> getUser1
package trace

import (
	"context"
	"fmt"
	"net/http"

	"github.com/yubo/apiserver/pkg/options"
	"github.com/yubo/apiserver/pkg/rest"
	"github.com/yubo/apiserver/pkg/tracing"
	"go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"
)

const (
	moduleName = "trace"
)

type Module struct {
	Name string
	ctx  context.Context
}

func New(ctx context.Context) *Module {
	return &Module{
		ctx: ctx,
	}
}

func (p *Module) Start() error {
	http, ok := options.APIServerFrom(p.ctx)
	if !ok {
		return fmt.Errorf("unable to get http server from the context")
	}

	p.installWs(http)
	return nil
}

func (p *Module) installWs(http rest.GoRestfulContainer) {
	rest.SwaggerTagRegister("tracing", "tracing demo")

	rest.WsRouteBuild(&rest.WsOption{
		Path:               "/tracing",
		Tags:               []string{"tracing"},
		GoRestfulContainer: http,
		Routes: []rest.WsRoute{
			{Method: "GET", SubPath: "v1/users/{name}", Handle: getUser},
			{Method: "GET", SubPath: "v2/users/{name}", Handle: getUser2},
		},
	})
}

type User struct {
	Name string `param:"path" name:"name"`
}

func getUser(w http.ResponseWriter, req *http.Request, in *User) (*User, error) {
	return getUser1(req.Context(), in)
}

func getUser1(ctx context.Context, in *User) (*User, error) {
	_, span := tracing.Start(ctx, "getUser1", oteltrace.WithAttributes(attribute.String("name", in.Name)))
	defer span.End()

	return in, nil
}

func getUser2(w http.ResponseWriter, req *http.Request, in *User) (*User, error) {
	ctx, span := tracing.Start(req.Context(), "getUser2", oteltrace.WithAttributes(attribute.String("name", in.Name)))
	defer span.End()

	return getUser1(ctx, in)
}
