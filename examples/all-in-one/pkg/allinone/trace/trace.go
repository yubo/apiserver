// this is a sample echo rest api module
//
//	/api/v1/users --> getUser1
//	/api/v2/users --> getUser2 --> getUser1
package trace

import (
	"context"
	"examples/all-in-one/pkg/allinone/config"
	"net/http"
	"time"

	"github.com/yubo/apiserver/components/dbus"
	"github.com/yubo/apiserver/pkg/rest"
	"github.com/yubo/apiserver/pkg/tracing"
	"go.opentelemetry.io/otel/attribute"
)

type trace struct {
	container rest.GoRestfulContainer
}

func New(ctx context.Context, cf *config.Config) *trace {
	return &trace{
		container: dbus.APIServer(),
	}
}

func (p *trace) Install() {
	rest.SwaggerTagRegister("tracing", "tracing demo")

	server.WsRouteBuild(&server.WsOption{
		Path:               "/tracing",
		Tags:               []string{"tracing"},
		GoRestfulContainer: p.container,
		Routes: []server.WsRoute{
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
	_, span := tracing.Start(ctx, "getUser1", attribute.String("name", in.Name))
	defer span.End(100 * time.Millisecond)

	return in, nil
}

func getUser2(w http.ResponseWriter, req *http.Request, in *User) (*User, error) {
	ctx, span := tracing.Start(req.Context(), "getUser2", attribute.String("name", in.Name))
	defer span.End(100 * time.Millisecond)

	return getUser1(ctx, in)
}
