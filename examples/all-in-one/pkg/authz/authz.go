// this is a sample authorization module
package authz

import (
	"context"
	"fmt"
	"net/http"

	"github.com/yubo/apiserver/pkg/options"
	"github.com/yubo/apiserver/pkg/rest"
	"k8s.io/klog/v2"
)

type authz struct {
	ctx context.Context
}

func New(ctx context.Context) *authz {
	return &authz{ctx: ctx}
}

func (p *authz) Start() error {
	http, ok := options.APIServerFrom(p.ctx)
	if !ok {
		return fmt.Errorf("unable to get http server from the context")
	}

	p.installWs(http)

	return nil
}

type AuthzInput struct {
	Namespace string `param:"path" name:"namespace"`
	Name      string `param:"path" name:"name"`
}
type AuthzBodyInput struct {
	Msg string `json:"msg"`
}

func (p *authz) installWs(http rest.GoRestfulContainer) {
	rest.SwaggerTagRegister("authorization", "authorization sample")

	rest.WsRouteBuild(&rest.WsOption{
		Path:               "/api/v1/namespaces/{namespace}/pods",
		Tags:               []string{"authorization"},
		GoRestfulContainer: http,
		Routes: []rest.WsRoute{{
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
