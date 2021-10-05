// this is a sample user rest api module
package user

import (
	"context"
	"fmt"
	"net/http"

	"github.com/yubo/apiserver/pkg/options"
	"github.com/yubo/apiserver/pkg/request"
	"github.com/yubo/apiserver/pkg/rest"
	"github.com/yubo/golib/orm"
	"k8s.io/klog/v2"
)

type Module struct {
	Name string
	http options.ApiServer
	db   orm.DB
	ctx  context.Context
}

func New(ctx context.Context) *Module {
	return &Module{
		ctx: ctx,
	}
}

func (p *Module) Start() error {
	var ok bool
	p.http, ok = options.ApiServerFrom(p.ctx)
	if !ok {
		return fmt.Errorf("unable to get http server from the context")
	}

	p.db, ok = options.DBFrom(p.ctx)
	if !ok {
		return fmt.Errorf("unable to get db from the context")
	}

	// init database
	if err := p.db.ExecRows([]byte(CREATE_TABLE_SQLITE)); err != nil {
		return err
	}

	p.installWs()

	addAuthScope()
	return nil
}

func (p *Module) installWs() {
	rest.SwaggerTagRegister("user", "user Api - for restful sample")

	rest.WsRouteBuild(&rest.WsOption{
		Path:               "/users",
		Produces:           []string{rest.MIME_JSON},
		Consumes:           []string{rest.MIME_JSON},
		Tags:               []string{"user"},
		GoRestfulContainer: p.http,
		Routes: []rest.WsRoute{{
			Method: "POST", SubPath: "/",
			Desc:   "create user",
			Handle: p.createUser,
		}, {
			Method: "GET", SubPath: "/",
			Desc:   "search/list users",
			Handle: p.getUsers,
		}, {
			Method: "GET", SubPath: "/{user-name}",
			Desc:   "get user",
			Handle: p.getUser,
		}, {
			Method: "PUT", SubPath: "/{user-name}",
			Desc:   "update user",
			Handle: p.updateUser,
		}, {
			Method: "DELETE", SubPath: "/{user-name}",
			Desc:   "delete user",
			Handle: p.deleteUser,
		}},
	})
}

func (p *Module) createUser(w http.ResponseWriter, req *http.Request, _ *rest.NoneParam, in *CreateUserInput) (*User, error) {
	return createUser(p.db, in)
}

func (p *Module) getUsers(w http.ResponseWriter, req *http.Request, param *GetUsersInput) (*GetUsersOutput, error) {
	user, _ := request.UserFrom(req.Context())
	klog.V(3).Infof("input %s user %+v", param, user)
	total, list, err := getUsers(p.db, param)

	return &GetUsersOutput{total, list}, err
}

func (p *Module) getUser(w http.ResponseWriter, req *http.Request, in *GetUserInput) (*User, error) {
	return getUser(p.db, in.Name)
}

func (p *Module) updateUser(w http.ResponseWriter, req *http.Request, param *UpdateUserParam, in *UpdateUserBody) (*User, error) {
	in.Name = param.Name
	return updateUser(p.db, in)
}

func (p *Module) deleteUser(w http.ResponseWriter, req *http.Request, in *DeleteUserInput) (*User, error) {
	return deleteUser(p.db, in.Name)
}

func addAuthScope() {
	rest.ScopeRegister("user:write", "user")
}
