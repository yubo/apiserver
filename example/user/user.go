// this is a sample user rest api module
package user

import (
	"fmt"
	"net/http"

	"github.com/emicklei/go-restful"
	"github.com/yubo/apiserver/pkg/options"
	"github.com/yubo/apiserver/pkg/openapi"
	"github.com/yubo/apiserver/pkg/request"
	"github.com/yubo/apiserver/pkg/responsewriters"
	"github.com/yubo/golib/orm"
	"github.com/yubo/golib/proc"
	"k8s.io/klog/v2"
)

const (
	moduleName = "user"
)

type Module struct {
	Name string
	http options.HttpServer
	//auth optioins.Auth
	db *orm.DB
}

var (
	_module = &Module{Name: moduleName}
	hookOps = []proc.HookOps{{
		Hook:     _module.start,
		Owner:    moduleName,
		HookNum:  proc.ACTION_START,
		Priority: proc.PRI_MODULE,
	}}
)

func (p *Module) start(ops *proc.HookOps) error {
	ctx := ops.Context()
	p.http = options.GenericServerMustFrom(ctx)
	p.db, _ = options.DBFrom(ctx)
	p.installWs()
	return nil
}

func (p *Module) installWs() {
	openapi.SwaggerTagRegister("user", "user Api - for restful sample")

	ws := new(restful.WebService)

	openapi.WsRouteBuild(&openapi.WsOption{
		Ws: ws.Path("/users").
			Produces(openapi.MIME_JSON).
			Consumes(openapi.MIME_JSON),
		Tags:      []string{"user"},
		RespWrite: respWrite,
	}, []openapi.WsRoute{{
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
	}})

	p.http.Add(ws)
}

func (p *Module) createUser(w http.ResponseWriter, req *http.Request, _ *openapi.NoneParam, in *CreateUserInput) (*CreateUserOutput, error) {
	id, err := createUser(p.db, in)

	return &CreateUserOutput{int64(id)}, err

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

func init() {
	proc.RegisterHooks(hookOps)
	addAuthScope()
}

func addAuthScope() {
	openapi.ScopeRegister("user:write", "user")
}

func respWrite(resp *restful.Response, data interface{}, err error) {
	var eMsg string
	code := int32(http.StatusOK)

	if err != nil {
		status := responsewriters.ErrorToAPIStatus(err)
		eMsg = status.Message
		code = status.Code

		if klog.V(3).Enabled() {
			klog.ErrorDepth(1, fmt.Sprintf("httpReturn %d %s", code, eMsg))
		}
	}

	resp.WriteEntity(map[string]interface{}{
		"dat":  data,
		"err":  eMsg,
		"code": code,
	})
}
