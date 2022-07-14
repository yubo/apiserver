package rest

import (
	"net/http"

	"github.com/emicklei/go-restful/v3"
	"github.com/go-openapi/spec"
	"github.com/yubo/apiserver/pkg/responsewriters"
	"k8s.io/klog/v2"
)

var (
	DefaultRespWriter = &defaultRespWriter{}
)

type RespWriter interface {
	// Name: use to register
	Name() string

	// AddRoute: register route
	AddRoute(method, path string)

	// SwaggerHandler: called at PostBuildSwaggerObjectHandler, use to rewrite the response definitions
	SwaggerHandler(s *spec.Swagger)

	// RespWrite: use to customize response output format
	RespWrite(resp *restful.Response, req *http.Request, data interface{}, err error)
}

type defaultRespWriter struct{}

// Name: use to register
func (p *defaultRespWriter) Name() string { return "rest.default" }

// AddRoute: register route
func (p *defaultRespWriter) AddRoute(method, path string) {}

// SwaggerHandler: called at PostBuildSwaggerObjectHandler, use to rewrite the response definitions
func (p *defaultRespWriter) SwaggerHandler(s *spec.Swagger) {}

// RespWrite: use to customize response output format
func (p *defaultRespWriter) RespWrite(resp *restful.Response, req *http.Request, data interface{}, err error) {
	if err != nil {
		code := responsewriters.Error(err, resp, req)
		klog.V(3).Infof("response %d %s", code, err.Error())
		return
	}

	switch t := data.(type) {
	case []byte:
		resp.Write(t)
	case *[]byte:
		resp.Write(*t)
	default:
		resp.WriteEntity(t)
	}
}
