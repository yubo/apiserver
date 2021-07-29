package options

import (
	"io"
	"net/http"

	"github.com/emicklei/go-restful"
	"github.com/yubo/apiserver/pkg/authentication/authenticator"
	"github.com/yubo/apiserver/pkg/authorization/authorizer"
)

type Client interface {
	GetId() string
	GetSecret() string
	GetRedirectUri() string
}

type ApiServer interface {
	Handle(string, http.Handler)
	HandleFunc(string, func(http.ResponseWriter, *http.Request))
	UnlistedHandle(string, http.Handler)
	UnlistedHandleFunc(string, func(http.ResponseWriter, *http.Request))
	Add(*restful.WebService) *restful.Container
	Filter(restful.FilterFunction)
	Address() string
	//Server() *server.GenericAPIServer
}

type Executer interface {
	Execute(wr io.Writer, data interface{}) error
}

type Authn interface {
	Authenticator() authenticator.Request
}

type Authz interface {
	Authorizer() authorizer.Authorizer
}
