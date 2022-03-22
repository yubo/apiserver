// v1 -> getUser1
// v2 -> getUser2 -> getUser1
// v3 -> getUser3 -> v1 -> getUser1
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/yubo/apiserver/pkg/cmdcli"
	"github.com/yubo/apiserver/pkg/options"
	"github.com/yubo/apiserver/pkg/rest"
	"github.com/yubo/apiserver/pkg/traces"
	"github.com/yubo/golib/cli"
	"github.com/yubo/golib/proc"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"

	server "github.com/yubo/apiserver/pkg/server/module"
	_ "github.com/yubo/apiserver/pkg/server/register"
	_ "github.com/yubo/apiserver/pkg/traces/register"
)

const (
	moduleName = "otel-traces.apiserver"
)

var (
	hookOps = []proc.HookOps{{
		Hook:     start,
		Owner:    moduleName,
		HookNum:  proc.ACTION_START,
		Priority: proc.PRI_MODULE,
	}}
)

func main() {
	command := proc.NewRootCmd(server.WithoutTLS(), proc.WithHooks(hookOps...))
	code := cli.Run(command)
	os.Exit(code)
}

func start(ctx context.Context) error {
	http, ok := options.APIServerFrom(ctx)
	if !ok {
		return fmt.Errorf("unable to get http server from the context")
	}

	installWs(http)
	return nil
}

func installWs(http rest.GoRestfulContainer) {
	rest.WsRouteBuild(&rest.WsOption{
		Path:               "/api",
		GoRestfulContainer: http,
		Routes: []rest.WsRoute{
			{Method: "GET", SubPath: "v1/users/{name}", Handle: getUser},
			{Method: "GET", SubPath: "v2/users/{name}", Handle: getUser2},
			{Method: "GET", SubPath: "v3/users/{name}", Handle: getUser3},
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
	_, span := traces.Start(ctx, "getUser1", oteltrace.WithAttributes(attribute.String("name", in.Name)))
	defer span.End()

	return in, nil
}

func getUser2(ctx context.Context, in *User) (*User, error) {
	ctx, span := traces.Start(ctx, "getUser2", oteltrace.WithAttributes(attribute.String("name", in.Name)))
	defer span.End()

	return getUser2(ctx, in)
}

func getUser3(w http.ResponseWriter, req *http.Request, in *User) (*User, error) {
	ctx, span := traces.Start(req.Context(), "getUser3", oteltrace.WithAttributes(attribute.String("name", in.Name)))
	defer span.End()

	return makeRequest(ctx, "127.0.0.1:8080", "/api/v1/users/"+in.Name)
}

func makeRequest(ctx context.Context, host, path string) (*User, error) {
	user := &User{}

	// Trace an HTTP client by wrapping the transport
	client := &http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}

	req, err := cmdcli.NewRequest(host,
		cmdcli.WithOutput(user),
		cmdcli.WithMethod("GET"),
		cmdcli.WithPrefix(path),
		cmdcli.WithClient(client),
		cmdcli.WithTraceInject(ctx),
	)
	if err != nil {
		return nil, err
	}

	if err := req.Do(ctx); err != nil {
		return nil, err
	}

	return user, nil
}
