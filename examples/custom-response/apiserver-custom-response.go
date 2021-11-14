package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/emicklei/go-restful"
	"github.com/yubo/apiserver/pkg/options"
	"github.com/yubo/apiserver/pkg/rest"
	"github.com/yubo/golib/logs"
	"github.com/yubo/golib/proc"
	"github.com/yubo/golib/util"

	_ "github.com/yubo/apiserver/pkg/rest/swagger/register"
	_ "github.com/yubo/apiserver/pkg/server/register"
)

// This example shows the minimal code needed to get a restful.WebService working.
//
// Open in browser http://localhost:8080/swagger

const (
	moduleName = "apiserver.swagger"
)

type User struct {
	Name     string  `json:"name"`
	NickName *string `json:"nickName"`
	Phone    *string `json:"phone"`
}

type GetUsersInput struct {
	rest.Pagination
	Query *string `param:"query" name:"query" description:"query user name or nick name"`
	Count bool    `param:"query" name:"count" description:"just response total count"`
}

type GetUsersOutput struct {
	Total int     `json:"total"`
	List  []*User `json:"list"`
}

type GetUsersOutputWrapper struct {
	Data GetUsersOutput `json:"data"`
	Err  string         `json:"err,omitempty"`
}

type GetUserInput struct {
	Name string `param:"path" name:"name" description:"query user name or nick name"`
}

type GetUserOutputWrapper struct {
	Data User   `json:"data"`
	Err  string `json:"err,omitempty"`
}

type Module struct{}

var (
	hookOps = []proc.HookOps{{
		Hook:     start,
		Owner:    moduleName,
		HookNum:  proc.ACTION_START,
		Priority: proc.PRI_MODULE,
	}}
	module Module
)

func main() {
	logs.InitLogs()
	defer logs.FlushLogs()

	proc.RegisterHooks(hookOps)

	if err := proc.NewRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

func start(ctx context.Context) error {
	http, ok := options.APIServerFrom(ctx)
	if !ok {
		return fmt.Errorf("unable to get http server from the context")
	}

	module.installWs(http)
	return nil
}

func respWrite(resp *restful.Response, req *http.Request, data interface{}, err error) {
	v := map[string]interface{}{"data": data}

	if err != nil {
		v["err"] = err.Error()
	}

	resp.WriteEntity(v)
}

func (p *Module) installWs(http rest.GoRestfulContainer) {
	rest.SwaggerTagRegister("user", "user Api - swagger api sample")
	rest.WsRouteBuild(&rest.WsOption{
		Path:               "/api/users",
		Produces:           []string{rest.MIME_JSON},
		Consumes:           []string{rest.MIME_JSON},
		Tags:               []string{"user"},
		GoRestfulContainer: http,
		RespWrite:          respWrite,
		Routes: []rest.WsRoute{{
			Method: "GET", SubPath: "/",
			Desc:   "search/list users",
			Handle: p.getUsers,
			Output: GetUsersOutputWrapper{},
		}, {
			Method: "GET", SubPath: "/{name}",
			Desc:   "get user",
			Handle: p.getUser,
			Output: GetUserOutputWrapper{},
		}},
	})
}

func (p *Module) getUsers(w http.ResponseWriter, req *http.Request, param *GetUsersInput) (*GetUsersOutput, error) {
	return &GetUsersOutput{
		Total: 1,
		List:  []*User{&User{Name: "tom", Phone: util.String("12345")}},
	}, nil
}

func (p *Module) getUser(w http.ResponseWriter, req *http.Request, in *GetUserInput) (*User, error) {

	return &User{Name: in.Name, Phone: util.String("12345")}, nil
}
