// this is a sample authentication module
package authn

import (
	"context"
	"examples/all-in-one/pkg/allinone/config"
	"net/http"

	"github.com/yubo/apiserver/components/dbus"
	"github.com/yubo/apiserver/pkg/authentication/user"
	"github.com/yubo/apiserver/pkg/request"
	"github.com/yubo/apiserver/pkg/rest"
	"github.com/yubo/apiserver/pkg/sessions"
	"github.com/yubo/golib/api/errors"
)

func New(ctx context.Context, cf *config.Config) *authn {
	return &authn{
		container: dbus.APIServer(),
	}
}

type authn struct {
	container    rest.GoRestfulContainer
	passwordfile dbus.PasswordfileT
}

func (p *authn) Install() {
	p.passwordfile = dbus.Passwordfile()

	rest.SwaggerTagRegister("authentication", "authentication sample")

	server.WsRouteBuild(&server.WsOption{
		Path:               "/authn",
		Tags:               []string{"authentication"},
		GoRestfulContainer: p.container,
		Routes: []server.WsRoute{{
			Method: "GET", SubPath: "/info",
			Desc:   "get authentication info",
			Handle: p.getAuthn,
			Acl:    "login",
			Scope:  "login",
		}, {
			Method: "POST", SubPath: "/login",
			Desc:   "authentication login, and set session userInfo",
			Handle: p.login,
		}, {
			Method: "POST", SubPath: "/logout",
			Desc:    "delete session userInfo",
			Consume: rest.MIME_ALL,
			Handle:  p.logout,
		}},
	})
}

func (p *authn) getAuthn(w http.ResponseWriter, req *http.Request) (*user.DefaultInfo, error) {
	u, ok := request.UserFrom(req.Context())
	if !ok {
		return nil, errors.NewUnauthorized("unable to get user info")
	}

	return &user.DefaultInfo{
		Name:   u.GetName(),
		UID:    u.GetUID(),
		Groups: u.GetGroups(),
		Extra:  u.GetExtra(),
	}, nil
}

type loginRequest struct {
	UserName string `json:"username"`
	Password string `json:"password"`
}

func (p *authn) login(w http.ResponseWriter, req *http.Request, r *loginRequest) (*user.DefaultInfo, error) {
	ctx := req.Context()

	u := p.passwordfile.Authenticate(ctx, r.UserName, r.Password)
	if u == nil {
		return nil, errors.NewUnauthorized("unable to get user info")
	}

	ret := &user.DefaultInfo{
		Name:   u.GetName(),
		UID:    u.GetUID(),
		Groups: u.GetGroups(),
		Extra:  u.GetExtra(),
	}

	sess := sessions.Default(ctx)
	sess.Set(sessions.UserInfoKey, ret)

	if err := sess.Save(); err != nil {
		return nil, errors.NewInternalError(err)
	}

	return ret, nil
}

func (p *authn) logout(w http.ResponseWriter, req *http.Request) (string, error) {
	sess := sessions.Default(req.Context())
	sess.Delete(sessions.UserInfoKey)
	if err := sess.Save(); err != nil {
		return "", errors.NewInternalError(err)
	}

	return "success logout", nil
}
