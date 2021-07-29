package rest

import (
	"fmt"
	"net/http"
	"reflect"

	"github.com/emicklei/go-restful"
	restfulspec "github.com/emicklei/go-restful-openapi"
	"github.com/yubo/golib/encoding/urlencoded"
	"k8s.io/klog/v2"
)

func WsRouteBuild(opt *WsOption) {
	if err := opt.build(); err != nil {
		panic(err)
	}

	if opt.GoRestfulContainer != nil {
		opt.GoRestfulContainer.Add(opt.Ws)
	}
}

type GoRestfulContainer interface {
	Add(*restful.WebService) *restful.Container
}

// sys.Filters > opt.Filter > opt.Filters > route.acl > route.filter > route.filters
type WsOption struct {
	Ws                 *restful.WebService
	Path               string
	Acl                func(aclName string) (restful.FilterFunction, string, error)
	Filter             restful.FilterFunction
	Filters            []restful.FilterFunction
	Produces           []string
	Consumes           []string
	PrefixPath         string
	Tags               []string
	Routes             []WsRoute
	RespWrite          func(resp *restful.Response, data interface{}, err error)
	GoRestfulContainer GoRestfulContainer
}

func (p *WsOption) Validate() error {
	if p.Ws == nil {
		p.Ws = &restful.WebService{}
	}
	if p.Path != "" {
		p.Ws = p.Ws.Path(p.Path)
	}
	if len(p.Produces) > 0 {
		p.Ws.Produces(p.Produces...)
	} else {
		p.Ws.Produces(MIME_JSON)
	}
	if len(p.Consumes) > 0 {
		p.Ws.Consumes(p.Consumes...)
	} else {
		p.Ws.Consumes(MIME_JSON)
	}
	return nil
}

func (p *WsOption) build() error {
	if err := p.Validate(); err != nil {
		return err
	}

	rb := NewRouteBuilder(p.Ws)

	for i := range p.Routes {
		route := &p.Routes[i]

		route.SubPath = p.PrefixPath + route.SubPath
		route.Filters = routeFilters(route, p)

		if route.Acl != "" {
			route.Desc += " acl(" + route.Acl + ")"
		}

		if route.Scope != "" {
			route.Desc += " scope(" + route.Scope + ")"
		}

		if route.Tags == nil && p.Tags != nil {
			route.Tags = p.Tags
		}

		if route.RespWrite == nil {
			if p.RespWrite != nil {
				route.RespWrite = p.RespWrite
			} else {
				route.RespWrite = RespWrite
			}
		}

		rb.Build(route)
	}
	return nil

}

type NoneParam struct{}

// opt.Filter > opt.Filters > route.acl > route.filter > route.filters
func routeFilters(route *WsRoute, opt *WsOption) (filters []restful.FilterFunction) {
	var filter restful.FilterFunction
	var err error

	if opt.Filter != nil {
		filters = append(filters, opt.Filter)
	}

	if len(opt.Filters) > 0 {
		filters = append(filters, opt.Filters...)
	}

	if route.Acl != "" && opt.Acl != nil {
		if filter, route.Scope, err = opt.Acl(route.Acl); err != nil {
			panic(err)
		}
		filters = append(filters, filter)
	}

	if route.Filter != nil {
		filters = append(filters, route.Filter)
	}

	if len(route.Filters) > 0 {
		filters = append(filters, route.Filters...)
	}

	return filters
}

type WsRoute struct {
	//Action string
	Acl string
	//--
	Method      string
	SubPath     string
	Desc        string
	Scope       string
	Consume     string
	Produce     string
	Handle      interface{}
	Filter      restful.FilterFunction
	Filters     []restful.FilterFunction
	ExtraOutput []ApiOutput
	Tags        []string
	RespWrite   func(resp *restful.Response, data interface{}, err error)
	// Input       interface{}
	// Output      interface{}
	// Handle      restful.RouteFunction
}

type ApiOutput struct {
	Code    int
	Message string
	Model   interface{}
}

// struct -> RouteBuilder do
type RouteBuilder struct {
	ws      *restful.WebService
	rb      *restful.RouteBuilder
	consume string
}

func NewRouteBuilder(ws *restful.WebService) *RouteBuilder {
	return &RouteBuilder{ws: ws}
}

func (p *RouteBuilder) Build(wr *WsRoute) error {
	var b *restful.RouteBuilder

	switch wr.Method {
	case "GET":
		b = p.ws.GET(wr.SubPath)
	case "POST":
		b = p.ws.POST(wr.SubPath)
	case "PUT":
		b = p.ws.PUT(wr.SubPath)
	case "DELETE":
		b = p.ws.DELETE(wr.SubPath)
	default:
		panic("unsupported method " + wr.Method)
	}
	p.rb = b

	if wr.Scope != "" {
		b.Metadata(SecurityDefinitionKey, wr.Scope)
	}

	if wr.Consume != "" {
		b.Consumes(wr.Consume)
	}

	if wr.Produce != "" {
		b.Produces(wr.Produce)
	}

	for _, filter := range wr.Filters {
		b.Filter(filter)
	}

	for _, out := range wr.ExtraOutput {
		b.Returns(out.Code, out.Message, out.Model)
	}

	if err := p.registerHandle(b, wr); err != nil {
		panic(err)
	}

	b.Doc(wr.Desc)
	b.Metadata(restfulspec.KeyOpenAPITags, wr.Tags)

	p.ws.Route(b)

	return nil
}

func noneHandle(req *restful.Request, resp *restful.Response) {}

func (p *RouteBuilder) registerHandle(b *restful.RouteBuilder, wr *WsRoute) error {
	if wr.Handle == nil {
		b.To(noneHandle)
		return nil
	}

	// handle(req *restful.Request, resp *restful.Response)
	// handle(req *restful.Request, resp *restful.Response, param *struct{})
	// handle(req *restful.Request, resp *restful.Response, param *struct{}, body []struct{})
	// handle(req *restful.Request, resp *restful.Response, param *struct{}, body *struct{})
	// handle(...) error
	// handle(...) (out struct{}, err error)

	// func (f HandlerFunc) ServeHTTP(w ResponseWriter, r *Request) {
	// handle(w ResponseWriter, r *Request, param *struct{}, body *struct{})

	v := reflect.ValueOf(wr.Handle)
	t := v.Type()

	numIn := t.NumIn()
	numOut := t.NumOut()

	if !((numIn == 2 || numIn == 3 || numIn == 4) && (numOut == 0 || numOut == 1 || numOut == 2)) {
		return fmt.Errorf("%s handle in num %d out num %d is Invalid", t.Name(), numIn, numOut)
	}

	if arg := t.In(0).String(); arg != "http.ResponseWriter" {
		panic(fmt.Sprintf("unable to get req http.ResponseWriter at in(0), get %s", arg))
	}

	if arg := t.In(1).String(); arg != "*http.Request" {
		panic(fmt.Sprintf("unable to get req *http.Request at in(1), get %s", arg))
	}

	var paramType reflect.Type
	var bodyType reflect.Type
	var useElem bool

	// 3, 4
	if numIn > 2 {
		paramType = t.In(2)

		switch paramType.Kind() {
		case reflect.Ptr:
			paramType = paramType.Elem()
			if paramType.Kind() != reflect.Struct {
				return fmt.Errorf("must ptr to struct, got ptr -> %s", paramType.Kind())
			}
		default:
			return fmt.Errorf("param just support ptr to struct")
		}

		p.buildParam(reflect.New(paramType).Elem().Interface())

	}

	// 4
	if numIn > 3 {
		bodyType = t.In(3)

		switch bodyType.Kind() {
		case reflect.Ptr:
			bodyType = bodyType.Elem()
			if bodyType.Kind() != reflect.Struct {
				return fmt.Errorf("must ptr to struct, got ptr -> %s", bodyType.Kind())
			}
		case reflect.Slice, reflect.Map:
			useElem = true
		default:
			return fmt.Errorf("just support slice and ptr to struct")
		}

		p.buildBody(wr.Consume, reflect.New(bodyType).Elem().Interface())
	}

	if numOut == 2 {
		ot := t.Out(0)
		if ot.Kind() == reflect.Ptr {
			ot = ot.Elem()
		}
		output := reflect.New(ot).Elem().Interface()
		b.Returns(http.StatusOK, "OK", output)
	}

	b.To(func(req *restful.Request, resp *restful.Response) {
		var ret []reflect.Value

		if numIn == 4 {
			// with param & body
			param := reflect.New(paramType).Interface()
			body := reflect.New(bodyType).Interface()

			if err := ReadEntity(req, param, body); err != nil {
				HttpWriteData(resp, nil, err)
				return
			}

			bodyValue := reflect.ValueOf(body)
			if useElem {
				bodyValue = bodyValue.Elem()
			}

			// TODO: use (w http.ResponseWriter, req *http.Request)
			ret = v.Call([]reflect.Value{
				reflect.ValueOf(resp.ResponseWriter),
				reflect.ValueOf(req.Request),
				reflect.ValueOf(param),
				bodyValue,
			})

		} else if numIn == 3 {
			// with param
			param := reflect.New(paramType).Interface()
			if err := ReadEntity(req, param, nil); err != nil {
				HttpWriteData(resp, nil, err)
				return
			}

			ret = v.Call([]reflect.Value{
				reflect.ValueOf(resp.ResponseWriter),
				reflect.ValueOf(req.Request),
				reflect.ValueOf(param),
			})
		} else {
			ret = v.Call([]reflect.Value{
				reflect.ValueOf(resp.ResponseWriter),
				reflect.ValueOf(req.Request),
			})
		}

		if numOut == 2 {
			wr.RespWrite(resp, vtoi(ret[0]), vtoe(ret[1]))
			return
		}

		if numOut == 1 {
			wr.RespWrite(resp, nil, vtoe(ret[0]))
		}
	})
	return nil
}

func (p *RouteBuilder) buildParam(param interface{}) *RouteBuilder {
	rv := reflect.Indirect(reflect.ValueOf(param))
	rt := rv.Type()

	if rv.Kind() != reflect.Struct || rt.String() == "time.Time" {
		panic(fmt.Sprintf("schema: interface must be a struct get %s %s", rt.Kind(), rt.String()))
	}

	fields := cachedTypeFields(rt)
	for _, f := range fields.list {
		if err := p.setParam(&f); err != nil {
			panic(err)
		}
	}
	return p
}

func (p *RouteBuilder) buildBody(consume string, body interface{}) *RouteBuilder {
	rv := reflect.Indirect(reflect.ValueOf(body))
	rt := rv.Type()

	klog.V(10).Infof("buildbody %s", rt.Name())
	if consume == MIME_URL_ENCODED {
		err := urlencoded.RouteBuilderReads(p.rb, rv)
		if err != nil {
			panic(err)
		}
		return p
	}

	p.rb.Reads(rv.Interface())
	return p
}

func (p *RouteBuilder) setParam(f *field) error {
	var parameter *restful.Parameter

	switch f.paramType {
	case PathType:
		parameter = restful.PathParameter(f.key, f.description)
	case QueryType:
		if f.hidden {
			return nil
		}
		parameter = restful.QueryParameter(f.key, f.description)
		parameter.Required(f.required)
	case HeaderType:
		if f.hidden {
			return nil
		}
		parameter = restful.HeaderParameter(f.key, f.description)
		parameter.Required(f.required)
	default:
		panicType(f.typ, "setParam")
	}

	switch f.typ.Kind() {
	case reflect.String:
		parameter.DataType("string")
	case reflect.Bool:
		parameter.DataType("boolean")
	case reflect.Uint, reflect.Int, reflect.Int32, reflect.Int64:
		parameter.DataType("integer")
	case reflect.Slice:
		if typeName := f.typ.Elem().Name(); typeName != "string" {
			panicType(f.typ, "unsupported param")
		}
	default:
		panicType(f.typ, "unsupported param")
	}

	if f.format != "" {
		parameter.DataFormat(f.format)
	}

	p.rb.Param(parameter)

	return nil
}

func vtoi(v reflect.Value) interface{} {
	if v.CanInterface() {
		return v.Interface()
	}

	return nil
}

func vtoe(v reflect.Value) error {
	if v.CanInterface() {
		e, _ := v.Interface().(error)
		return e
	}

	return nil
}
