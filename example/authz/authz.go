// this is a sample authorization module
package authz

import (
	"context"
	"fmt"
	"net/http"

	"github.com/emicklei/go-restful"
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
	http, ok := options.ApiServerFrom(p.ctx)
	if !ok {
		return fmt.Errorf("unable to get http server from the context")
	}

	p.installWs(http)

	return nil
}

type AuthzInput struct {
	Namespace string `param:"path" name:"namespace"`
	Name      string `param:"path" name:"authz-name"`
}
type AuthzBodyInput struct {
	Msg string `json:"msg"`
}

func (p *authz) installWs(http options.ApiServer) {
	rest.SwaggerTagRegister("authorization", "authorization sample")

	ws := new(restful.WebService)

	rest.WsRouteBuild(&rest.WsOption{
		Ws: ws.Path("/api/v1/namespaces/{namespace}/authz").
			Produces(rest.MIME_JSON).
			Consumes(rest.MIME_JSON),
		Tags: []string{"authorization"},
	}, []rest.WsRoute{{
		Method: "GET", SubPath: "/{authz-name}",
		Desc:   "get namespace info",
		Handle: p.ns,
	}, {
		Method: "POST", SubPath: "/{authz-name}",
		Desc:   "create namespace",
		Handle: p.nsbody,
	}, {
		Method: "DELETE", SubPath: "/{authz-name}",
		Desc:   "delete namespace info",
		Handle: p.ns,
	}, {
		Method: "PUT", SubPath: "/{authz-name}",
		Desc:   "update namespace",
		Handle: p.nsbody,
	}})

	http.Add(ws)
}

func (p *authz) ns(w http.ResponseWriter, req *http.Request, in *AuthzInput) (string, error) {
	klog.Infof("http authz %s %s", req.Method, in.Namespace)
	return fmt.Sprintf("%s %s", req.Method, in.Namespace), nil
}

func (p *authz) nsbody(w http.ResponseWriter, req *http.Request, in *AuthzInput, body *AuthzBodyInput) (string, error) {
	klog.Infof("http authz %s %s %s", req.Method, in.Namespace, body.Msg)
	return fmt.Sprintf("%s %s %s", req.Method, in.Namespace, body.Msg), nil
}
