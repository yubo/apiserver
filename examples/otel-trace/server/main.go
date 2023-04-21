// /api/v1/users --> getUser1
// /api/v2/users --> getUser2 --> getUser1
// /api/v3/users --> getUser3 --> /api/v1/users
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/yubo/apiserver/components/cli"
	"github.com/yubo/apiserver/pkg/client"
	"github.com/yubo/apiserver/pkg/proc"
	"github.com/yubo/apiserver/pkg/proc/options"
	"github.com/yubo/apiserver/pkg/rest"
	"github.com/yubo/apiserver/pkg/tracing"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"

	server "github.com/yubo/apiserver/pkg/server/module"
	_ "github.com/yubo/apiserver/pkg/server/register"
	_ "github.com/yubo/apiserver/pkg/tracing/register"
)

func main() {
	cmd := proc.NewRootCmd(server.WithoutTLS(), proc.WithRun(start))
	code := cli.Run(cmd)
	os.Exit(code)
}

func start(ctx context.Context) error {
	srv, ok := options.APIServerFrom(ctx)
	if !ok {
		return fmt.Errorf("unable to get http server from the context")
	}

	rest.WsRouteBuild(&rest.WsOption{
		Path:               "/api",
		GoRestfulContainer: srv,
		Routes: []rest.WsRoute{
			{Method: "GET", SubPath: "v1/users/{name}", Handle: getUser},
			{Method: "GET", SubPath: "v2/users/{name}", Handle: getUser2},
			{Method: "GET", SubPath: "v3/users/{name}", Handle: getUser3},
		},
	})

	return nil
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

func getUser3(w http.ResponseWriter, req *http.Request, in *User) (*User, error) {
	ctx, span := tracing.Start(req.Context(), "getUser3", oteltrace.WithAttributes(attribute.String("name", in.Name)))
	defer span.End()

	return makeRequest(ctx, "127.0.0.1:8080", "/api/v1/users/"+in.Name)
}

func makeRequest(ctx context.Context, host, path string) (*User, error) {
	user := &User{}

	// Trace an HTTP client by wrapping the transport
	c := &http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}

	req, err := client.NewRequest(host,
		client.WithOutput(user),
		client.WithMethod("GET"),
		client.WithPrefix(path),
		client.WithClient(c),
	)
	if err != nil {
		return nil, err
	}

	if err := req.Do(ctx); err != nil {
		return nil, err
	}

	return user, nil
}
