// this is a sample user rest api module
package user

import (
	"context"
	"fmt"
	"net/http"

	"github.com/yubo/apiserver/pkg/options"
	"github.com/yubo/apiserver/pkg/rest"
	"github.com/yubo/apiserver/pkg/server"

	_ "github.com/yubo/apiserver/pkg/models/register"
)

type Module struct {
	Name string
	UserModel
	ctx context.Context
}

func New(ctx context.Context) *Module {
	return &Module{
		ctx: ctx,
	}
}

func (p *Module) Start() error {
	http, ok := options.APIServerFrom(p.ctx)
	if !ok {
		return fmt.Errorf("unable to get API server from the context")
	}

	p.UserModel = NewUser()

	p.installWs(http)

	addAuthScope()
	return nil
}

func (p *Module) installWs(http server.APIServer) {
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
			Handle: p.update,
		}, {
			Method: "DELETE", SubPath: "/{name}",
			Desc:   "delete user",
			Handle: p.delete,
		}},
	})
}

func (p *Module) create(w http.ResponseWriter, req *http.Request, _ *rest.NonParam, in *CreateUserInput) (*User, error) {
	return p.Create(req.Context(), in.User())
}

func (p *Module) get(w http.ResponseWriter, req *http.Request, in *GetUserInput) (*User, error) {
	return p.Get(req.Context(), in.Name)
}

func (p *Module) list(w http.ResponseWriter, req *http.Request, in *ListInput) (ret *ListUserOutput, err error) {
	ret = &ListUserOutput{}

	opts, err := in.ListOptions(in.Query, &ret.Total)
	if err != nil {
		return nil, err
	}

	ret.List, err = p.List(req.Context(), *opts)
	return ret, err
}

func (p *Module) update(w http.ResponseWriter, req *http.Request, param *UpdateUserParam, in *UpdateUserInput) (*User, error) {
	in.Name = param.Name
	return p.Update(req.Context(), in)
}

func (p *Module) delete(w http.ResponseWriter, req *http.Request, in *DeleteUserInput) (*User, error) {
	return p.Delete(req.Context(), in.Name)
}

func addAuthScope() {
	rest.ScopeRegister("user:write", "user")
}
