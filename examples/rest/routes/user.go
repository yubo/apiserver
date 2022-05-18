package routes

import (
	"net/http"

	"examples/rest/api"
	"examples/rest/models"

	"github.com/yubo/apiserver/pkg/rest"
)

type user struct {
	models.User
}

func InstallUser(http rest.GoRestfulContainer) {
	user := &user{models.NewUser()}
	rest.WsRouteBuild(&rest.WsOption{
		Path:               "/api",
		GoRestfulContainer: http,
		Routes: []rest.WsRoute{
			{Method: "POST", SubPath: "/users", Handle: user.create},
			{Method: "GET", SubPath: "/users", Handle: user.list},
			{Method: "GET", SubPath: "/users/{name}", Handle: user.get},
			{Method: "PUT", SubPath: "/users/{name}", Handle: user.update},
			{Method: "DELETE", SubPath: "/users/{name}", Handle: user.delete},
		},
	})

}

func (p *user) create(w http.ResponseWriter, req *http.Request, _ *rest.NonParam, in *api.CreateUserInput) (*api.User, error) {
	return p.Create(req.Context(), in.User())
}

func (p *user) get(w http.ResponseWriter, req *http.Request, in *api.GetUserInput) (*api.User, error) {
	return p.Get(req.Context(), in.Name)
}

func (p *user) list(w http.ResponseWriter, req *http.Request, in *api.ListInput) (ret *api.ListUserOutput, err error) {
	ret = &api.ListUserOutput{}

	opts, err := in.ListOptions(in.Query, &ret.Total)
	if err != nil {
		return nil, err
	}

	ret.List, err = p.List(req.Context(), *opts)
	return ret, err
}

func (p *user) update(w http.ResponseWriter, req *http.Request, param *api.UpdateUserParam, in *api.UpdateUserInput) (*api.User, error) {
	in.Name = param.Name
	return p.Update(req.Context(), in)
}

func (p *user) delete(w http.ResponseWriter, req *http.Request, in *api.DeleteUserInput) (*api.User, error) {
	return p.Delete(req.Context(), in.Name)
}
