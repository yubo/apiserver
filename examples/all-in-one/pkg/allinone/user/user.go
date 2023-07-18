// this is a sample user rest api module
package user

import (
	"context"
	"net/http"

	"github.com/yubo/apiserver/components/dbus"
	"github.com/yubo/apiserver/pkg/rest"
	libapi "github.com/yubo/golib/api"

	"examples/all-in-one/pkg/allinone/config"
	"examples/all-in-one/pkg/api"
	"examples/all-in-one/pkg/filters"
	"examples/all-in-one/pkg/models"
)

func New(ctx context.Context, cf *config.Config) *user {
	return &user{
		container: dbus.APIServer(),
		user:      models.NewUser(),
	}
}

type user struct {
	container rest.GoRestfulContainer
	user      *models.User
}

func (p *user) Install() {
	rest.ScopeRegister("user:write", "user")
	rest.SwaggerTagRegister("user", "user Api - for restful sample")

	server.WsRouteBuild(&server.WsOption{
		Path:               "/users",
		Produces:           []string{rest.MIME_JSON},
		Consumes:           []string{rest.MIME_JSON},
		Tags:               []string{"user"},
		GoRestfulContainer: p.container,
		Routes: []server.WsRoute{{
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
			Handle: p.del,
		}},
	})
}

func (p *user) create(w http.ResponseWriter, req *http.Request, in *createInput) error {
	return p.user.Create(req.Context(), in.User())
}

func (p *user) get(w http.ResponseWriter, req *http.Request, in *nameParam) (*api.User, error) {
	return p.user.Get(req.Context(), in.Name)
}

func (p *user) list(w http.ResponseWriter, req *http.Request, in *listParam) (ret *listOutput, err error) {
	ret = &listOutput{}

	opts, err := in.GetListOptions(in.Query, &ret.Total)
	if err != nil {
		return nil, err
	}

	ret.List, err = p.user.List(req.Context(), opts)
	return ret, err
}

func (p *user) update(w http.ResponseWriter, req *http.Request, param *nameParam, in *updateInput) error {
	in.Name = &param.Name
	return p.user.Update(req.Context(), in.User())
}

func (p *user) del(w http.ResponseWriter, req *http.Request, in *nameParam) error {
	return p.user.Delete(req.Context(), in.Name)
}

type createInput struct {
	Name string
	Age  int
}

func (p *createInput) User() *api.User {
	return &api.User{
		Name: &p.Name,
		Age:  &p.Age,
	}
}

type nameParam struct {
	Name string `param:"path"`
}

type listParam struct {
	libapi.PageParams
	Query string `param:"query" description:"query user"`
}

type listOutput struct {
	List  []api.User `json:"list"`
	Total int        `json:"total"`
}

type updateInput struct {
	Name *string `json:"-"` // from UpdateUserParam
	Age  *int    `json:"age"`
}

func (p *updateInput) User() *api.User {
	return &api.User{
		Name: p.Name,
		Age:  p.Age,
	}
}
