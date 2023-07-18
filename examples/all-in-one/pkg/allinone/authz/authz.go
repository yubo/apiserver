// this is a sample authorization module
package authz

import (
	"context"
	"examples/all-in-one/pkg/allinone/config"
	"fmt"
	"net/http"

	"github.com/yubo/apiserver/components/dbus"
	"github.com/yubo/apiserver/pkg/rest"
	"k8s.io/klog/v2"
)

func New(ctx context.Context, cf *config.Config) *authz {
	return &authz{
		container: dbus.APIServer(),
	}
}

type authz struct {
	container rest.GoRestfulContainer
}

func (p *authz) Install() {
	rest.SwaggerTagRegister("authorization", "authorization sample")

	server.WsRouteBuild(&server.WsOption{
		Path:               "/api/v1/namespaces/{namespace}/pods",
		Tags:               []string{"authorization"},
		GoRestfulContainer: p.container,
		Routes: []server.WsRoute{{
			Method: "GET", SubPath: "/{name}",
			Desc:   "get pod info",
			Handle: p.ns,
		}, {
			Method: "POST", SubPath: "/{name}",
			Desc:   "create pod",
			Handle: p.nsbody,
		}, {
			Method: "DELETE", SubPath: "/{name}",
			Desc:   "delete pod",
			Handle: p.ns,
		}, {
			Method: "PUT", SubPath: "/{name}",
			Desc:   "update pod",
			Handle: p.nsbody,
		}},
	})
}

func (p *authz) ns(w http.ResponseWriter, req *http.Request, in *AuthzInput) (string, error) {
	klog.Infof("http authz %s %s", req.Method, in.Namespace)
	return fmt.Sprintf("%s %s", req.Method, in.Namespace), nil
}

func (p *authz) nsbody(w http.ResponseWriter, req *http.Request, in *AuthzInput, body *AuthzBodyInput) (string, error) {
	klog.Infof("http authz %s %s %s", req.Method, in.Namespace, body.Msg)
	return fmt.Sprintf("%s %s %s", req.Method, in.Namespace, body.Msg), nil
}

type AuthzInput struct {
	Namespace string `param:"path" name:"namespace"`
	Name      string `param:"path" name:"name"`
}
type AuthzBodyInput struct {
	Msg string `json:"msg"`
}
