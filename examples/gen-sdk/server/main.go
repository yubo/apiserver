package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/yubo/apiserver/components/cli"
	"github.com/yubo/apiserver/pkg/proc"
	v1 "github.com/yubo/apiserver/pkg/proc/api/v1"
	"github.com/yubo/apiserver/pkg/proc/options"
	"github.com/yubo/apiserver/pkg/rest"
	"github.com/yubo/golib/api"
	"github.com/yubo/golib/api/errors"
	"github.com/yubo/golib/util"

	server "github.com/yubo/apiserver/pkg/server/module"
	_ "github.com/yubo/apiserver/pkg/server/register"
)

const (
	moduleName = "gen-sdk.example"
	lang       = "python"
)

type User struct {
	Name     string  `json:"name"`
	NickName *string `json:"nickName"`
	Phone    *string `json:"phone"`
}

type CreateUserInput struct {
	Name     string  `json:"name"`
	NickName *string `json:"nickName"`
	Phone    *string `json:"phone"`
}

type CreateUserOutput User

type GetUsersInput struct {
	api.PageParams
	Query *string `param:"query" name:"query" description:"query user"`
	Count bool    `param:"query" name:"count" description:"just response total count"`
}

func (p *GetUsersInput) Validate() error {
	return nil
}

func (p GetUsersInput) String() string {
	return util.Prettify(p)
}

type GetUsersOutput struct {
	Total int     `json:"total"`
	List  []*User `json:"list"`
}

type GetUserInput struct {
	Name string `param:"path" name:"user-name"`
}

func (p *GetUserInput) Validate() error {
	return nil
}

type UpdateUserParam struct {
	Name string `param:"path" name:"user-name"`
}

type UpdateUserBody struct {
	Name     string  `json:"-" sql:",where"`
	NickName *string `json:"nickName"`
	Phone    *string `json:"phone"`
}

type DeleteUserInput struct {
	Name string `param:"path" name:"user-name"`
}

type DeleteUserOutput User

type Module struct {
	users []*User
}

var (
	hookOps = []v1.HookOps{{
		Hook:     start,
		Owner:    moduleName,
		HookNum:  v1.ACTION_START,
		Priority: v1.PRI_MODULE,
	}}
	module Module
)

func main() {
	command := proc.NewRootCmd(server.WithoutTLS(), proc.WithHooks(hookOps...))
	code := cli.Run(command)
	os.Exit(code)
}

func start(ctx context.Context) error {
	server, ok := options.APIServerFrom(ctx)
	if !ok {
		return fmt.Errorf("unable to get API server from the context")
	}

	module.installWs(server)
	return nil
}

func (p *Module) installWs(http rest.GoRestfulContainer) {
	rest.SwaggerTagRegister("user", "user Api - swagger api sample")
	rest.WsRouteBuild(&rest.WsOption{
		Path:               "/api/user",
		Produces:           []string{rest.MIME_JSON},
		Consumes:           []string{rest.MIME_JSON},
		Tags:               []string{"user"},
		GoRestfulContainer: http,
		Routes: []rest.WsRoute{{
			Method: "POST", SubPath: "/",
			Desc:      "create user",
			Operation: "createUser",
			Handle:    p.createUser,
		}, {
			Method: "GET", SubPath: "/",
			Desc:      "search/list users",
			Operation: "getUsers",
			Handle:    p.getUsers,
		}, {
			Method: "GET", SubPath: "/{user-name}",
			Desc:      "get user",
			Operation: "getUser",
			Handle:    p.getUser,
		}, {
			Method: "PUT", SubPath: "/{user-name}",
			Desc:      "update user",
			Operation: "updateUser",
			Handle:    p.updateUser,
		}, {
			Method: "DELETE", SubPath: "/{user-name}",
			Desc:      "delete user",
			Operation: "deleteUser",
			Handle:    p.deleteUser,
		}},
	})
}

func (p *Module) createUser(w http.ResponseWriter, req *http.Request, in *CreateUserInput) (CreateUserOutput, error) {
	user := User{
		Name:     in.Name,
		NickName: in.NickName,
		Phone:    in.Phone,
	}

	p.users = append(p.users, &user)

	return CreateUserOutput(user), nil
}

func (p *Module) getUsers(w http.ResponseWriter, req *http.Request, param *GetUsersInput) (*GetUsersOutput, error) {
	return &GetUsersOutput{Total: len(p.users), List: p.users}, nil
}

func (p *Module) getUser(w http.ResponseWriter, req *http.Request, in *GetUserInput) (*User, error) {
	for _, u := range p.users {
		if u.Name == in.Name {
			return u, nil
		}
	}

	return nil, errors.NewNotFound("user")
}

func (p *Module) updateUser(w http.ResponseWriter, req *http.Request, param *UpdateUserParam, in *UpdateUserBody) (*User, error) {
	in.Name = param.Name
	for _, u := range p.users {
		if u.Name == in.Name {
			if in.NickName != nil {
				u.NickName = in.NickName
			}
			if in.Phone != nil {
				u.Phone = in.Phone
			}
			return u, nil
		}
	}

	return nil, errors.NewNotFound("user")
}

func (p *Module) deleteUser(w http.ResponseWriter, req *http.Request, in *DeleteUserInput) (*User, error) {
	for i, u := range p.users {
		if u.Name == in.Name {
			p.users[i] = p.users[len(p.users)-1]
			p.users = p.users[:len(p.users)-1]
		}
		return u, nil
	}

	return nil, errors.NewNotFound("user")
}
