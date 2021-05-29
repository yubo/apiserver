// this is a sample echo rest api module
package session

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/emicklei/go-restful"
	"github.com/yubo/apiserver/example/user"
	"github.com/yubo/apiserver/pkg/openapi"
	"github.com/yubo/apiserver/pkg/options"
	"github.com/yubo/golib/net/session"
)

type module struct {
	Name    string
	http    options.GenericServer
	session session.SessionManager
	ctx     context.Context
}

func New(ctx context.Context) *module {
	return &module{
		ctx: ctx,
	}
}

func (p *module) Start() error {
	var ok bool

	p.http, ok = options.GenericServerFrom(p.ctx)
	if !ok {
		return fmt.Errorf("unable to get http server from the context")
	}

	p.session, ok = options.SessionManagerFrom(p.ctx)
	if !ok {
		return fmt.Errorf("unable to get session manager from the context")
	}

	db, ok := options.DBFrom(p.ctx)
	if !ok {
		return fmt.Errorf("unable to get db")
	}

	if err := db.ExecRows([]byte(session.CREATE_TABLE_SQLITE)); err != nil {
		return err
	}
	if err := db.ExecRows([]byte(user.CREATE_TABLE_SQLITE)); err != nil {
		return err
	}

	p.installWs()

	return nil
}

func (p *module) installWs() {
	openapi.SwaggerTagRegister("session", "demo session")

	ws := new(restful.WebService)

	openapi.WsRouteBuild(&openapi.WsOption{
		Ws:     ws.Path("/session").Produces(openapi.MIME_JSON).Consumes("*/*"),
		Filter: Filter(p.session),
		Tags:   []string{"session"},
	}, []openapi.WsRoute{{
		Method: "GET", SubPath: "/",
		Desc:   "get session info",
		Handle: p.info,
	}, {
		Method: "GET", SubPath: "/set",
		Desc:   "set session info",
		Handle: p.set,
	}, {
		Method: "GET", SubPath: "/reset",
		Desc:   "reset session info",
		Handle: p.reset,
	}})

	p.http.Add(ws)
}

// show session information
func (p *module) info(w http.ResponseWriter, req *http.Request) (string, error) {
	sess, ok := session.SessionFrom(req.Context())
	if !ok {
		return "can't get session info", nil
	}

	userName := sess.Get("userName")
	if userName == "" {
		return "can't get username from session", nil
	}

	cnt, err := strconv.Atoi(sess.Get("info.cnt"))
	if err != nil {
		cnt = 0
	}

	cnt++
	sess.Set("info.cnt", strconv.Itoa(cnt))
	return fmt.Sprintf("%s %d", userName, cnt), nil
}

// set session
func (p *module) set(w http.ResponseWriter, req *http.Request) (string, error) {
	sess, ok := session.SessionFrom(req.Context())
	if ok {
		sess.Set("userName", "tom")
	}
	return "set username successfully", nil
}

// reset session
func (p *module) reset(w http.ResponseWriter, req *http.Request) (string, error) {
	sess, ok := session.SessionFrom(req.Context())
	if ok {
		sess.Reset()
	}
	return "reset successfully", nil
}

func Filter(manager session.SessionManager) restful.FilterFunction {
	return func(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
		sess, err := manager.Start(resp, req.Request)
		if err != nil {
			openapi.HttpWriteErr(resp, fmt.Errorf("session start err %s", err))
			return
		}
		ctx := session.WithSession(req.Request.Context(), sess)
		req.Request.WithContext(ctx)

		chain.ProcessFilter(req, resp)

		sess.Update(resp)
	}
}
