package user

import (
	"context"
	"net/http"

	"examples/rest/api"
	"examples/rest/models"

	"github.com/yubo/apiserver/components/dbus"
	"github.com/yubo/apiserver/pkg/server"
	"github.com/yubo/apiserver/plugin/responsewriter/umi"
	libapi "github.com/yubo/golib/api"
)

func New(ctx context.Context) *user {
	return &user{
		server: dbus.APIServer(),
		user:   models.NewUser(),
	}
}

type user struct {
	server *server.GenericAPIServer
	user   *models.User
}

func (p *user) Install() {
	server.SwaggerTagRegister("v1", "user Api - for restful sample")
	server.SwaggerTagRegister("v2", "user Api(umi styles) - for restful sample - https://pro.ant.design/zh-CN/docs/request")

	server.WsRouteBuild(&server.WsOption{
		Path:     "/api/v1",
		Produces: []string{server.MIME_JSON},
		Consumes: []string{server.MIME_JSON},
		Tags:     []string{"v1"},
		Server:   p.server,
		Routes: []server.WsRoute{{
			Method:    "POST",
			SubPath:   "/users",
			Operation: "createUser",
			Desc:      "create User",
			Handle:    p.create,
		}, {
			Method:    "GET",
			SubPath:   "/users",
			Operation: "listUser",
			Desc:      "list User",
			Handle:    p.list,
		}, {
			Method:     "GET",
			SubPath:    "/user/{name}",
			Operation:  "getUserByName",
			Desc:       "get user by name",
			Deprecated: true,
			Handle:     p.get,
		}, {
			Method:    "GET",
			SubPath:   "/users/{name}",
			Desc:      "get user by name",
			Operation: "getUser",
			Handle:    p.get,
		}, {
			Method:    "PUT",
			SubPath:   "/users/{name}",
			Desc:      "update user by name",
			Operation: "updateUser",
			Handle:    p.update,
		}, {
			Method:    "DELETE",
			SubPath:   "/users/{name}",
			Desc:      "delete user by name",
			Operation: "deleteUser",
			Handle:    p.del,
		}},
	})

	server.WsRouteBuild(&server.WsOption{
		Path:       "/api/v2",
		Produces:   []string{server.MIME_JSON},
		Consumes:   []string{server.MIME_JSON},
		Tags:       []string{"v2"},
		RespWriter: umi.RespWriter,
		Server:     p.server,
		Routes: []server.WsRoute{{
			Method:    "POST",
			SubPath:   "/users",
			Operation: "createUserV2",
			Desc:      "create user",
			Handle:    p.create,
		}, {
			Method:    "GET",
			SubPath:   "/users",
			Operation: "listUserV2",
			Desc:      "list user",
			Handle:    p.list,
		}, {
			Method:     "GET",
			SubPath:    "/user/{name}",
			Operation:  "getUserByNameV2",
			Desc:       "get user by name",
			Deprecated: true,
			Handle:     p.get,
		}, {
			Method:    "GET",
			SubPath:   "/users/{name}",
			Operation: "getUserV2",
			Desc:      "get user by name",
			Handle:    p.get,
		}, {
			Method:    "PUT",
			SubPath:   "/users/{name}",
			Operation: "updateUserV2",
			Desc:      "update user",
			Handle:    p.update,
		}, {
			Method:    "DELETE",
			SubPath:   "/users/{name}",
			Operation: "deleteUserV2",
			Desc:      "delete user",
			Handle:    p.del,
		}},
	})
}

func (p *user) create(w http.ResponseWriter, req *http.Request, in *createInput) error {
	return p.user.Create(req.Context(), in.User())
}

func (p *user) get(w http.ResponseWriter, req *http.Request, in *nameParam) (*api.User, error) {
	return p.user.Get(req.Context(), in.Name)
}

// default styles
func (p *user) list(w http.ResponseWriter, req *http.Request, in *listParam) (ret *listOutput, err error) {
	ret = &listOutput{}

	opts, err := in.GetListOptions(in.Query, &ret.Total)
	if err != nil {
		return nil, err
	}

	ret.List, err = p.user.List(req.Context(), *opts)
	return
}

func (p *user) update(w http.ResponseWriter, req *http.Request, param *nameParam, in *updateInput) error {
	in.Name = &param.Name
	return p.user.Update(req.Context(), in.User())
}

func (p *user) del(w http.ResponseWriter, req *http.Request, in *nameParam) error {
	return p.user.Delete(req.Context(), in.Name)
}

type createInput struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
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
