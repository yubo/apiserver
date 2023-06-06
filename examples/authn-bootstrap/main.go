package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/yubo/apiserver/components/cli"
	"github.com/yubo/apiserver/components/dbus"
	"github.com/yubo/apiserver/pkg/authentication/user"
	bootstrapapi "github.com/yubo/apiserver/pkg/cluster-bootstrap/token/api"
	"github.com/yubo/apiserver/pkg/models"
	"github.com/yubo/apiserver/pkg/proc"
	"github.com/yubo/apiserver/pkg/request"
	"github.com/yubo/apiserver/pkg/rest"
	"github.com/yubo/golib/api"
	"k8s.io/klog/v2"

	// api server
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

const (
	moduleName = "bootstrap.authn.exmaples"

	// Fake values for testing.
	tokenID     = "foobar"           // 6 letters
	tokenSecret = "circumnavigation" // 16 letters
)

func main() {
	cmd := proc.NewRootCmd(proc.WithRun(start))
	code := cli.Run(cmd)
	os.Exit(code)
}

func start(ctx context.Context) error {
	srv, err := dbus.GetAPIServer()
	if err != nil {
		return err
	}
	rest.WsRouteBuild(&rest.WsOption{
		Path:               "/hello",
		GoRestfulContainer: srv,
		Routes: []rest.WsRoute{
			{Method: "GET", SubPath: "/", Handle: hw},
		},
	})

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

func hw(w http.ResponseWriter, req *http.Request) (*user.DefaultInfo, error) {
	klog.InfoS("recv", "req.header", req.Header)
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
