// this is a sample authentication module
package authn

import (
	"context"
	"fmt"
	"net/http"

	"github.com/yubo/apiserver/pkg/authentication/user"
	"github.com/yubo/apiserver/pkg/options"
	"github.com/yubo/apiserver/pkg/request"
	"github.com/yubo/apiserver/pkg/rest"
)

type authn struct {
	ctx context.Context
}

func New(ctx context.Context) *authn {
	return &authn{ctx: ctx}
}

func (p *authn) Start() error {
	http, ok := options.APIServerFrom(p.ctx)
	if !ok {
		return fmt.Errorf("unable to get http server from the context")
	}

	p.installWs(http)

	return nil
}

func (p *authn) installWs(http rest.GoRestfulContainer) {
	rest.SwaggerTagRegister("authentication", "authentication sample")

	rest.WsRouteBuild(&rest.WsOption{
		Path:               "/authn",
		Tags:               []string{"authentication"},
		GoRestfulContainer: http,
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
