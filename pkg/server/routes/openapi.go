package routes

import (
	"github.com/emicklei/go-restful/v3"
	"github.com/go-openapi/spec"
	"github.com/yubo/apiserver/pkg/rest"
)

type OpenAPI struct{}

func (p OpenAPI) Install(apiPath string, container *restful.Container, infoProps spec.InfoProps, securitySchemes []rest.SchemeConfig) error {
	return rest.InstallApiDocs(apiPath, container, infoProps, securitySchemes)
}
