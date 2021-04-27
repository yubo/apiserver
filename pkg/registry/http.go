package registry

import (
	"fmt"
	"net/http"

	"github.com/emicklei/go-restful"
)

const (
	_ uint16 = iota << 3
	PRI_AUTHN
	PRI_AUTHZ
	PRI_SWAGGER
	PRI_GRPC
	PRI_TRACING
)

var (
	Register register
)

type register struct {
	http httpRegister
}

type httpRegister struct {
	Handles       map[string]http.Handler
	UnlistHandles map[string]http.Handler
}

func Handle(pattern string, handler http.Handler) error {
	if _, ok := Register.http.Handles[pattern]; ok {
		return fmt.Errorf("http handler %s already exists", pattern)
	}
	Register.http.Handles[pattern] = handler
	return nil
}
func HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) error {
	return Handle(pattern, http.HandlerFunc(handler))
}

func UnlistHandle(pattern string, handler http.Handler) error {
	if _, ok := Register.http.UnlistHandles[pattern]; ok {
		return fmt.Errorf("http handler %s already exists", pattern)
	}
	Register.http.UnlistHandles[pattern] = handler
	return nil
}
func UnlistHandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) error {
	return UnlistHandle(pattern, http.HandlerFunc(handler))
}

func Addws(service *restful.WebService) *restful.Container {
}
