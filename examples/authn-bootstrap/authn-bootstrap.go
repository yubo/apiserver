package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/yubo/apiserver/pkg/authentication/user"
	bootstrapapi "github.com/yubo/apiserver/pkg/cluster-bootstrap/token/api"
	"github.com/yubo/apiserver/pkg/models"
	"github.com/yubo/apiserver/pkg/options"
	"github.com/yubo/apiserver/pkg/request"
	"github.com/yubo/apiserver/pkg/rest"
	"github.com/yubo/golib/api"
	"github.com/yubo/golib/cli"
	"github.com/yubo/golib/proc"

	// api server
	server "github.com/yubo/apiserver/pkg/server/module"
	_ "github.com/yubo/apiserver/pkg/server/register"

	// authn
	_ "github.com/yubo/apiserver/pkg/authentication/register"
	_ "github.com/yubo/apiserver/plugin/authenticator/token/bootstrap/register"

	// for models
	_ "github.com/yubo/apiserver/pkg/db/register"
	_ "github.com/yubo/apiserver/pkg/models/register"
	_ "github.com/yubo/golib/orm/mysql"
	_ "github.com/yubo/golib/orm/sqlite"
)

// go run ./authn-bootstrap.go --db-driver=sqlite3 --db-dsn="file:test.db?cache=shared&mode=memory"
// $curl -Ss  -H 'Authorization: bearer foobar.circumnavigation' http://localhost:8080/hello
// {
//  "Name": "system:bootstrap:foobar",
//  "UID": "",
//  "Groups": [
//   "system:bootstrappers",
//   "system:bootstrappers:foo",
//   "system:authenticated"
//  ],
//  "Extra": null
// }

const (
	moduleName = "example.bootstrap.authn"

	// Fake values for testing.
	tokenID     = "foobar"           // 6 letters
	tokenSecret = "circumnavigation" // 16 letters
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
	command := proc.NewRootCmd(
		server.WithoutTLS(),
		proc.WithHooks(hookOps...),
	)
	code := cli.Run(command)
	os.Exit(code)
}

func start(ctx context.Context) error {
	http, ok := options.APIServerFrom(ctx)
	if !ok {
		return fmt.Errorf("unable to get http server from the context")
	}
	installWs(http)

	// create secret
	secret := models.NewSecret()
	secret.Create(ctx, &api.Secret{
		ObjectMeta: api.ObjectMeta{
			Name: bootstrapapi.BootstrapTokenSecretPrefix + tokenID,
		},
		Data: map[string][]byte{
			bootstrapapi.BootstrapTokenIDKey:               []byte(tokenID),
			bootstrapapi.BootstrapTokenSecretKey:           []byte(tokenSecret),
			bootstrapapi.BootstrapTokenUsageAuthentication: []byte("true"),
			bootstrapapi.BootstrapTokenExtraGroupsKey:      []byte("system:bootstrappers:foo"),
		},
		Type: "bootstrap.kubernetes.io/token",
	})
	return nil
}

func installWs(http rest.GoRestfulContainer) {
	rest.WsRouteBuild(&rest.WsOption{
		Path:               "/hello",
		GoRestfulContainer: http,
		Routes: []rest.WsRoute{
			{Method: "GET", SubPath: "/", Handle: hw},
		},
	})
}

func hw(w http.ResponseWriter, req *http.Request) (*user.DefaultInfo, error) {
	u, ok := request.UserFrom(req.Context())
	if !ok {
		return nil, fmt.Errorf("unable to get user info")
	}
	return &user.DefaultInfo{
		Name:   u.GetName(),
		UID:    u.GetUID(),
		Groups: u.GetGroups(),
		Extra:  u.GetExtra(),
	}, nil
}
