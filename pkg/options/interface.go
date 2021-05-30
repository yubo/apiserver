package options

import (
	"io"
	"net/http"

	"github.com/emicklei/go-restful"
	"github.com/yubo/apiserver/pkg/authentication/authenticator"
	"github.com/yubo/apiserver/pkg/authorization/authorizer"
	//genericoptions "github.com/yubo/apiserver/pkg/apiserver/options"
	//"github.com/yubo/apiserver/pkg/apiserver/server"
)

type Client interface {
	GetId() string
	GetSecret() string
	GetRedirectUri() string
}

type HttpServer interface {
	// http
	Handle(pattern string, handler http.Handler)
	HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request))

	// restful.Container
	Add(service *restful.WebService) *restful.Container
	Filter(filter restful.FilterFunction)
}

type GenericServer interface {
	Handle(string, http.Handler)
	HandleFunc(string, func(http.ResponseWriter, *http.Request))
	UnlistedHandle(string, http.Handler)
	UnlistedHandleFunc(string, func(http.ResponseWriter, *http.Request))
	Add(*restful.WebService) *restful.Container
	Filter(restful.FilterFunction)
	//Server() *server.GenericAPIServer
}

type Executer interface {
	Execute(wr io.Writer, data interface{}) error
}

type SecureServing interface {
	//SecureServingInfo() *server.SecureServingInfo
	//Config() *genericoptions.SecureServingOptions
}

type Authn interface {
	//APIAudiences() authenticator.Audiences
	Authenticator() authenticator.Request
}

type Authz interface {
	Authorizer() authorizer.Authorizer
}
