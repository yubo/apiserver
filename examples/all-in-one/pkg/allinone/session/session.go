// this is a sample echo rest api module
package session

import (
	"context"
	"examples/all-in-one/pkg/allinone/config"
	"fmt"
	"net/http"

	"github.com/yubo/apiserver/components/dbus"
	"github.com/yubo/apiserver/pkg/rest"

	"github.com/yubo/apiserver/pkg/sessions"
	_ "github.com/yubo/apiserver/pkg/sessions/register"
)

func New(ctx context.Context, cf *config.Config) *session {
	return &session{
		container: dbus.APIServer(),
	}

}

type session struct {
	container rest.GoRestfulContainer
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
			Method: "GET", SubPath: "/clear",
			Desc:   "clear session info",
			Handle: p.clear,
		}},
	})
}

// show session information
func (p *session) info(w http.ResponseWriter, req *http.Request) (string, error) {
	sess := sessions.Default(req.Context())

	userName, _ := sess.Get("username").(string)
	if userName == "" {
		return "can't get username from session", nil
	}

	cnt, _ := sess.Get("infocnt").(int)
	cnt++
	sess.Set("infocnt", cnt)
	sess.Save()

	return fmt.Sprintf("%d hi, %s", cnt, userName), nil
}

// set session
func (p *session) set(w http.ResponseWriter, req *http.Request) (string, error) {
	sess := sessions.Default(req.Context())

	sess.Set("username", "tom")
	sess.Save()

	return "set username successfully", nil
}

// clear session
func (p *session) clear(w http.ResponseWriter, req *http.Request) (string, error) {
	sess := sessions.Default(req.Context())
	sess.Clear()
	sess.Save()
	return "reset successfully", nil
}
