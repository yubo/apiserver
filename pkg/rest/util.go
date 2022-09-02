package rest

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/emicklei/go-restful/v3"
	"github.com/go-openapi/spec"
	"github.com/yubo/golib/runtime"
	"github.com/yubo/golib/scheme"
)

type Validator interface {
	Validate() error
}

// TODO: remove
func IsEmptyValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Ptr:
		return v.IsNil()
	}
	return false
}

func OperationFrom(s *spec.Swagger, method, path string) (*spec.Operation, error) {
	p, err := s.Paths.JSONLookup(strings.TrimRight(path, "/"))
	if err != nil {
		return nil, err
	}

	var ret *spec.Operation
	pathItem := p.(*spec.PathItem)

	switch strings.ToLower(method) {
	case "get":
		ret = pathItem.Get
	case "post":
		ret = pathItem.Post
	case "patch":
		ret = pathItem.Patch
	case "delete":
		ret = pathItem.Delete
	case "put":
		ret = pathItem.Put
	case "head":
		ret = pathItem.Head
	case "options":
		ret = pathItem.Options
	default:
		// unsupported method
		return nil, fmt.Errorf("skipping Security openapi spec for %s:%s, unsupported method '%s'", method, path, method)
	}

	if ret == nil {
		return nil, fmt.Errorf("can't found %s:%s", method, path)
	}

	return ret, nil
}

func sqlOrder(order string) string {
	switch strings.ToLower(order) {
	case "ascend", "asc":
		return "ASC"
	case "descend", "desc":
		return "DESC"
	default:
		return "ASC"
	}
}

func newInterface(rt reflect.Type) interface{} {
	if rt == nil {
		return nil
	}

	return reflect.New(rt).Interface()
}

func newInterfaceFromInterface(i interface{}) interface{} {
	rt := reflect.ValueOf(i).Type()
	if rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
	}

	return reflect.New(rt).Interface()
}

//var _ GoRestfulContainer = new(BaseContainer)

type BaseContainer struct {
	*restful.Container
	runtime.NegotiatedSerializer
}

func (p *BaseContainer) Remove(*restful.WebService) error                       { return nil }
func (p *BaseContainer) UnlistedHandle(path string, handler http.Handler)       {}
func (p *BaseContainer) HandlePrefix(path string, handler http.Handler)         {}
func (p *BaseContainer) UnlistedHandlePrefix(path string, handler http.Handler) {}
func (p *BaseContainer) Serializer() runtime.NegotiatedSerializer               { return p.NegotiatedSerializer }

func NewBaseContainer() *BaseContainer {
	ret := &BaseContainer{}
	ret.Container = restful.NewContainer()
	ret.NegotiatedSerializer = scheme.NegotiatedSerializer
	return ret
}
