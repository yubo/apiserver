// this is a sample echo rest api module
package session

import (
	"context"
	"examples/all-in-one/pkg/allinone/config"
	"fmt"
	"net/http"
	"strconv"

	"github.com/yubo/apiserver/pkg/options"
	"github.com/yubo/apiserver/pkg/request"
	"github.com/yubo/apiserver/pkg/rest"
	"github.com/yubo/apiserver/pkg/session/types"

	_ "github.com/yubo/apiserver/pkg/session/register"
)

func New(ctx context.Context, cf *config.Config) *session {
	return &session{
		container: options.APIServerMustFrom(ctx),
		session:   options.SessionManagerMustFrom(ctx),
	}

}

type session struct {
	container rest.GoRestfulContainer
	session   types.SessionManager
}

func (p *session) Install() {
	rest.SwaggerTagRegister("session", "demo session")
	rest.WsRouteBuild(&rest.WsOption{
		// << set filter >>
		// has been added filters.session at apiserver.DefaultBuildHandlerChain
		// Filter: filters.Session(p.session),
		Path:               "/session",
		Consumes:           []string{"*/*"},
		Tags:               []string{"session"},
		GoRestfulContainer: p.container,
		Routes: []rest.WsRoute{{
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
		}},
	})
}

// show session information
func (p *session) info(w http.ResponseWriter, req *http.Request) (string, error) {
	sess, ok := request.SessionFrom(req.Context())
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
	return fmt.Sprintf("%d hi, %s", cnt, userName), nil
}

// set session
func (p *session) set(w http.ResponseWriter, req *http.Request) (string, error) {
	sess, ok := request.SessionFrom(req.Context())
	if ok {
		sess.Set("userName", "tom")
		return "set username successfully", nil
	}
	return "can't get session", nil
}

// reset session
func (p *session) reset(w http.ResponseWriter, req *http.Request) (string, error) {
	sess, ok := request.SessionFrom(req.Context())
	if ok {
		sess.Reset()
	}
	return "reset successfully", nil
}
