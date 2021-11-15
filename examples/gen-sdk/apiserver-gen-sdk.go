package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"

	"github.com/yubo/apiserver/pkg/cmdcli"
	"github.com/yubo/apiserver/pkg/options"
	"github.com/yubo/apiserver/pkg/rest"
	"github.com/yubo/golib/api/errors"
	"github.com/yubo/golib/configer"
	"github.com/yubo/golib/logs"
	"github.com/yubo/golib/proc"
	"github.com/yubo/golib/util"

	server "github.com/yubo/apiserver/pkg/server/module"
	_ "github.com/yubo/apiserver/pkg/server/register"
)

// https://github.com/openapitools/openapi-generator
// https://openapi-generator.tech/docs/generators
// go run apiserver-gen-sdk.go

const (
	moduleName = "example.gensdk.openapi"
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
	rest.Pagination
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
	hookOps = []proc.HookOps{{
		Hook:     start,
		Owner:    moduleName,
		HookNum:  proc.ACTION_START,
		Priority: proc.PRI_MODULE,
	}, {
		Hook:     postStart,
		Owner:    moduleName,
		HookNum:  proc.ACTION_START,
		Priority: proc.PRI_SYS_POSTSTART,
	}}
	module Module
)

func main() {
	logs.InitLogs()
	defer logs.FlushLogs()

	proc.RegisterHooks(hookOps)

	if err := server.NewRootCmdWithoutTLS().Execute(); err != nil {
		os.Exit(1)
	}
}

func start(ctx context.Context) error {
	server, ok := options.APIServerFrom(ctx)
	if !ok {
		return fmt.Errorf("unable to get API server from the context")
	}

	module.installWs(server)
	return nil
}

func postStart(ctx context.Context) error {
	defer proc.Shutdown()

	server, ok := options.APIServerFrom(ctx)
	if !ok {
		return fmt.Errorf("unable to get API server from the context")
	}

	fd, err := os.Create("./apidocs.json")
	if err != nil {
		return err
	}
	defer fd.Close()

	req, err := cmdcli.NewRequestWithConfig(
		server.Config().LoopbackClientConfig,
		cmdcli.WithPrefix("/apidocs.json"),
		cmdcli.WithOutput(fd))
	if err != nil {
		return err
	}

	if err := req.Do(context.Background()); err != nil {
		return err
	}

	pwd, err := os.Getwd()
	if err != nil {
		return err
	}

	cmd := exec.Command("docker", "run", "--rm",
		"-v", pwd+":/local",
		"openapitools/openapi-generator-cli",
		"generate",
		"-i", "/local/apidocs.json",
		"-g", lang,
		"-o", "/local/out/"+lang)

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
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

func (p *Module) createUser(w http.ResponseWriter, req *http.Request, _ *rest.NonParam, in *CreateUserInput) (CreateUserOutput, error) {
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
