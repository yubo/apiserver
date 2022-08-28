// this is a sample authentication module
package authn

import (
	"context"
	"examples/all-in-one/pkg/allinone/config"
	"fmt"
	"net/http"

	"github.com/yubo/apiserver/pkg/authentication/user"
	"github.com/yubo/apiserver/pkg/options"
	"github.com/yubo/apiserver/pkg/request"
	"github.com/yubo/apiserver/pkg/rest"
)

func New(ctx context.Context, cf *config.Config) *authn {
	return &authn{
		container: options.APIServerMustFrom(ctx),
	}
}

type authn struct {
	container rest.GoRestfulContainer
}

func (p *authn) Install() {
	rest.SwaggerTagRegister("authentication", "authentication sample")

	rest.WsRouteBuild(&rest.WsOption{
		Path:               "/authn",
		Tags:               []string{"authentication"},
		GoRestfulContainer: p.container,
		Routes: []rest.WsRoute{{
			Method: "GET", SubPath: "/",
			Desc:   "get authentication info",
			Handle: p.getAuthn,
			Acl:    "read",
			Scope:  "read",
		}},
	})
}

func (p *authn) getAuthn(w http.ResponseWriter, req *http.Request) (*user.DefaultInfo, error) {
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
