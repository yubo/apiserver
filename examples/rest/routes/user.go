package routes

import (
	"net/http"

	"examples/rest/api"
	"examples/rest/models"

	"github.com/yubo/apiserver/pkg/rest"
	"github.com/yubo/apiserver/plugin/responsewriter/umi"
)

type route struct {
	models.User
}

func InstallUser(http rest.GoRestfulContainer) {
	r := &route{models.NewUser()}

	rest.SwaggerTagRegister("v1", "user Api - for restful sample")
	rest.SwaggerTagRegister("v2", "user Api(umi styles) - for restful sample - https://pro.ant.design/zh-CN/docs/request")

	rest.WsRouteBuild(&rest.WsOption{
		Path:               "/api/v1",
		Produces:           []string{rest.MIME_JSON},
		Consumes:           []string{rest.MIME_JSON},
		Tags:               []string{"v1"},
		GoRestfulContainer: http,
		Routes: []rest.WsRoute{{
			Method:    "POST",
			SubPath:   "/users",
			Operation: "createUser",
			Desc:      "create User",
			Handle:    r.createUser,
		}, {
			Method:    "GET",
			SubPath:   "/users",
			Operation: "listUser",
			Desc:      "list User",
			Handle:    r.listUser,
		}, {
			Method:     "GET",
			SubPath:    "/user/{name}",
			Operation:  "getUserByName",
			Desc:       "get user by name",
			Deprecated: true,
			Handle:     r.getUser,
		}, {
			Method:    "GET",
			SubPath:   "/users/{name}",
			Desc:      "get user by name",
			Operation: "getUser",
			Handle:    r.getUser,
		}, {
			Method:    "PUT",
			SubPath:   "/users/{name}",
			Desc:      "update user by name",
			Operation: "updateUser",
			Handle:    r.updateUser,
		}, {
			Method:    "DELETE",
			SubPath:   "/users/{name}",
			Desc:      "delete user by name",
			Operation: "deleteUser",
			Handle:    r.deleteUser,
		}},
	})

	rest.WsRouteBuild(&rest.WsOption{
		Path:               "/api/v2",
		Produces:           []string{rest.MIME_JSON},
		Consumes:           []string{rest.MIME_JSON},
		Tags:               []string{"v2"},
		RespWriter:         umi.RespWriter,
		GoRestfulContainer: http,
		Routes: []rest.WsRoute{{
			Method:    "POST",
			SubPath:   "/users",
			Operation: "createUserV2",
			Desc:      "create user",
			Handle:    r.createUser,
		}, {
			Method:    "GET",
			SubPath:   "/users",
			Operation: "listUserV2",
			Desc:      "list user",
			Handle:    r.listUser,
		}, {
			Method:     "GET",
			SubPath:    "/user/{name}",
			Operation:  "getUserByNameV2",
			Desc:       "get user by name",
			Deprecated: true,
			Handle:     r.getUser,
		}, {
			Method:    "GET",
			SubPath:   "/users/{name}",
			Operation: "getUserV2",
			Desc:      "get user by name",
			Handle:    r.getUser,
		}, {
			Method:    "PUT",
			SubPath:   "/users/{name}",
			Operation: "updateUserV2",
			Desc:      "update user",
			Handle:    r.updateUser,
		}, {
			Method:    "DELETE",
			SubPath:   "/users/{name}",
			Operation: "deleteUserV2",
			Desc:      "delete user",
			Handle:    r.deleteUser,
		}},
	})
}

func (p *route) createUser(w http.ResponseWriter, req *http.Request, _ *rest.NonParam, in *api.CreateUserInput) (*api.User, error) {
	return p.Create(req.Context(), in.User())
}

func (p *route) getUser(w http.ResponseWriter, req *http.Request, in *api.GetUserInput) (*api.User, error) {
	return p.Get(req.Context(), in.Name)
}

// default styles
func (p *route) listUser(w http.ResponseWriter, req *http.Request, in *api.ListInput) (ret *api.ListUserOutput, err error) {
	ret = &api.ListUserOutput{}

	var total int64
	opts, err := in.ListOptions(in.Query, &total)
	if err != nil {
		return nil, err
	}

	list, err := p.List(req.Context(), *opts)
	if err != nil {
		return nil, err
	}

	return &api.ListUserOutput{
		List:        list,
		CurrentPage: in.GetCurPage(),
		PageSize:    in.GetPageSize(),
		Total:       total,
	}, nil
}

func (p *route) updateUser(w http.ResponseWriter, req *http.Request, param *api.UpdateUserParam, in *api.UpdateUserInput) (*api.User, error) {
	in.Name = param.Name
	return p.Update(req.Context(), in)
}

func (p *route) deleteUser(w http.ResponseWriter, req *http.Request, in *api.DeleteUserInput) (*api.User, error) {
	return p.Delete(req.Context(), in.Name)
}
