// this is a sample user rest api module
package user

import (
	"context"
	"fmt"
	"net/http"

	"github.com/yubo/apiserver/pkg/options"
	"github.com/yubo/apiserver/pkg/rest"
	"github.com/yubo/apiserver/pkg/server"

	"examples/all-in-one/pkg/api"
	"examples/all-in-one/pkg/filters"
	"examples/all-in-one/pkg/models"

	_ "github.com/yubo/apiserver/pkg/models/register"
)

type User struct {
	Name string
	user *models.User
	ctx  context.Context
}

func New(ctx context.Context) *User {
	return &User{
		ctx: ctx,
	}
}

func (p *User) Start() error {
	http, ok := options.APIServerFrom(p.ctx)
	if !ok {
		return fmt.Errorf("unable to get API server from the context")
	}

	p.user = models.NewUser()

	p.installWs(http)

	addAuthScope()
	return nil
}

func (p *User) installWs(http server.APIServer) {
	rest.SwaggerTagRegister("user", "user Api - for restful sample")

	rest.WsRouteBuild(&rest.WsOption{
		Path:               "/users",
		Produces:           []string{rest.MIME_JSON},
		Consumes:           []string{rest.MIME_JSON},
		Tags:               []string{"user"},
		GoRestfulContainer: http,
		Routes: []rest.WsRoute{{
			Method: "POST", SubPath: "/",
			Desc:   "create user",
			Handle: p.create,
		}, {
			Method: "GET", SubPath: "/",
			Desc:   "search/list users",
			Handle: p.list,
		}, {
			Method: "GET", SubPath: "/{name}",
			Desc:   "get user",
			Handle: p.get,
		}, {
			Method: "PUT", SubPath: "/{name}",
			Desc:   "update user",
			Filter: filters.WithTx,
			Handle: p.update,
		}, {
			Method: "DELETE", SubPath: "/{name}",
			Desc:   "delete user",
			Handle: p.delete,
		}},
	})
}

func (p *User) create(w http.ResponseWriter, req *http.Request, _ *rest.NonParam, in *api.CreateUserInput) error {
	return p.user.Create(req.Context(), in.User())
}

func (p *User) get(w http.ResponseWriter, req *http.Request, in *api.GetUserParam) (*api.User, error) {
	return p.user.Get(req.Context(), in.Name)
}

func (p *User) list(w http.ResponseWriter, req *http.Request, in *api.ListInput) (ret *api.ListUserOutput, err error) {
	ret = &api.ListUserOutput{}

	opts, err := in.ListOptions(in.Query, &ret.Total)
	if err != nil {
		return nil, err
	}

	ret.List, err = p.user.List(req.Context(), *opts)
	return ret, err
}

func (p *User) update(w http.ResponseWriter, req *http.Request, param *api.UpdateUserParam, in *api.UpdateUserInput) error {
	in.Name = param.Name
	return p.user.Update(req.Context(), in)
}

func (p *User) delete(w http.ResponseWriter, req *http.Request, in *api.DeleteUserParam) error {
	return p.user.Delete(req.Context(), in.Name)
}

func addAuthScope() {
	rest.ScopeRegister("user:write", "user")
}
