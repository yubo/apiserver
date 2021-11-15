package routes

import (
	"github.com/emicklei/go-restful"
	"github.com/go-openapi/spec"
	"github.com/yubo/apiserver/pkg/rest"
)

type OpenAPI struct{}

func (p OpenAPI) Install(apiPath string, container *restful.Container,
	infoProps spec.InfoProps, securitySchemes []rest.SchemeConfig) error {

	rest.InstallApiDocs(apiPath, container, infoProps, securitySchemes)
	return nil
}
