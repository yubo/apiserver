package rest

import (
	"fmt"
	"net/http"
	"path"
	"reflect"
	"strings"

	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	"github.com/emicklei/go-restful/v3"
	"github.com/go-openapi/spec"
	"github.com/yubo/apiserver/pkg/audit"
	"github.com/yubo/apiserver/pkg/metrics"
	"github.com/yubo/apiserver/pkg/request"
	"github.com/yubo/apiserver/pkg/responsewriters"
	"github.com/yubo/apiserver/pkg/rest/urlencoded"
	"k8s.io/klog/v2"
)

var (
	ScopeCatalog          = map[string]string{}
	securitySchemeCatalog = map[string]*spec.SecurityScheme{}
	respWriterCatalog     = map[string]RespWriter{}
	swaggerTags           = []spec.Tag{}
	DefaultContentTypes   = []string{MIME_ALL, MIME_JSON}
)

func WsRouteBuild(opt *WsOption) {
	if err := opt.build(); err != nil {
		panic(err)
	}

	if opt.GoRestfulContainer != nil {
		opt.GoRestfulContainer.Add(opt.Ws)
	} else {
		klog.Warningf("unable to get restful.Container, routebuild %s skiped", opt.Path)
	}
}

type GoRestfulContainer interface {
	// Add a WebService to the Container. It will detect duplicate root paths and exit in that case.
	Add(*restful.WebService) *restful.Container
	// Remove a WebService from the Container.
	Remove(service *restful.WebService) error
	// Handle registers the handler for the given pattern.
	// If a handler already exists for pattern, Handle panics.
	Handle(path string, handler http.Handler)
	// UnlistedHandle registers the handler for the given pattern, but doesn't list it.
	// If a handler already exists for pattern, Handle panics.
	UnlistedHandle(path string, handler http.Handler)
	// HandlePrefix is like Handle, but matches for anything under the path.  Like a standard golang trailing slash.
	HandlePrefix(path string, handler http.Handler)
	// UnlistedHandlePrefix is like UnlistedHandle, but matches for anything under the path.  Like a standard golang trailing slash.
	UnlistedHandlePrefix(path string, handler http.Handler)
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
	RespWriter         RespWriter
	GoRestfulContainer GoRestfulContainer
	ParameterCodec     request.ParameterCodec
}

func (p *WsOption) Validate() error {
	if p.Ws == nil {
		p.Ws = &restful.WebService{}
	}
	if p.Path != "" {
		p.Ws = p.Ws.Path(p.Path)
	}
	if p.Ws.RootPath() == "/" {
		klog.Warningf("rootpath is set to /, which may overwrite the existing route")
	}
	if len(p.Produces) > 0 {
		p.Ws.Produces(p.Produces...)
	} else {
		p.Ws.Produces(DefaultContentTypes...)
	}
	if len(p.Consumes) > 0 {
		p.Ws.Consumes(p.Consumes...)
	} else {
		p.Ws.Consumes(DefaultContentTypes...)
	}
	if p.ParameterCodec == nil {
		p.ParameterCodec = NewParameterCodec()
	}
	return nil
}

func (p *WsOption) build() error {
	if err := p.Validate(); err != nil {
		return err
	}

	if p.RespWriter == nil {
		p.RespWriter = DefaultRespWriter
	}

	rb := NewRouteBuilder(p.Ws, p.ParameterCodec)

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

		if route.RespWriter == nil {
			route.RespWriter = p.RespWriter
		}

		if err := rb.Build(route); err != nil {
			return err
		}
	}
	return nil

}

type NonParam struct{}

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
	Method  string
	SubPath string
	Desc    string
	Scope   string
	Consume string
	Produce string

	// Operation allows you to document what the actual method/function call is of the Route.
	// Unless called, the operation name is derived from the RouteFunction set using To(..).
	Operation string

	// Notes is a verbose explanation of the operation behavior. Optional.
	Notes string

	Deprecated bool

	// handle(req *restful.Request, resp *restful.Response)
	// handle(req *restful.Request, resp *restful.Response, param *struct{})
	// handle(req *restful.Request, resp *restful.Response, param *struct{}, body *slice)
	// handle(req *restful.Request, resp *restful.Response, param *struct{}, body *map)
	// handle(req *restful.Request, resp *restful.Response, param *struct{}, body *struct)
	Handle interface{}

	Filter      restful.FilterFunction
	Filters     []restful.FilterFunction
	ExtraOutput []ApiOutput
	Tags        []string
	RespWriter  RespWriter
	InputParam  interface{} // pri > handle
	InputBody   interface{} // pri > handle
	Output      interface{} // pri > handle
}

type ApiOutput struct {
	Code    int
	Message string
	Model   interface{}
}

// struct -> RouteBuilder do
type RouteBuilder struct {
	ws             *restful.WebService
	rb             *restful.RouteBuilder
	consume        string
	parameterCodec request.ParameterCodec
}

func NewRouteBuilder(ws *restful.WebService, codec request.ParameterCodec) *RouteBuilder {
	return &RouteBuilder{ws: ws, parameterCodec: codec}
}

func (p *RouteBuilder) Build(wr *WsRoute) error {
	var b *restful.RouteBuilder

	switch strings.ToUpper(wr.Method) {
	case "GET", "LIST":
		b = p.ws.GET(wr.SubPath)
	case "POST", "CREATE":
		b = p.ws.POST(wr.SubPath)
	case "PUT", "UPDATE":
		b = p.ws.PUT(wr.SubPath)
	case "DELETE":
		b = p.ws.DELETE(wr.SubPath)
	default:
		panic("unsupported method " + wr.Method)
	}
	p.rb = b

	if wr.Deprecated {
		b.Deprecate()
	}

	if wr.Scope != "" {
		b.Metadata(SecurityDefinitionKey, wr.Scope)
	}

	if wr.Consume != "" {
		b.Consumes(wr.Consume)
	}

	if wr.Produce != "" {
		b.Produces(wr.Produce)
	}

	if wr.Operation != "" {
		b.Operation(wr.Operation)
	}

	if wr.Notes != "" {
		b.Notes(wr.Notes)
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
	// handle(req *restful.Request, resp *restful.Response, param *struct{}, body *slice)
	// handle(req *restful.Request, resp *restful.Response, param *struct{}, body *map)
	// handle(req *restful.Request, resp *restful.Response, param *struct{}, body *struct)
	// handle(...) error
	// handle(...) (out *struct{}, err error)

	// func (f HandlerFunc) ServeHTTP(w ResponseWriter, r *Request) {
	// handle(w ResponseWriter, r *Request, param *struct{}, body *struct{})

	v := reflect.ValueOf(wr.Handle)
	t := v.Type()

	numIn := t.NumIn()
	numOut := t.NumOut()

	if !((numIn >= 2 && numIn <= 4) && numOut <= 2) {
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

	// build input param
	if numIn > 2 {
		inputParam := wr.InputParam
		paramType = t.In(2)

		switch paramType.Kind() {
		case reflect.Ptr:
			paramType = paramType.Elem()
			if paramType.Kind() != reflect.Struct {
				return fmt.Errorf("param must ptr to struct, got ptr -> %s", paramType.Kind())
			}
		default:
			return fmt.Errorf("param just support ptr to struct")
		}

		if wr.InputParam == nil {
			inputParam = reflect.New(paramType).Elem().Interface()
		}

		if inputParam != nil {
			p.buildParam(inputParam)
		}
	}

	// build intput body
	if numIn > 3 {
		inputBody := wr.InputBody
		bodyType = t.In(3)

		if bodyType.Kind() != reflect.Ptr {
			return fmt.Errorf("payload must be a ptr, got %s", bodyType.Kind())
		}
		bodyType = bodyType.Elem()

		switch bodyType.Kind() {
		case reflect.Struct, reflect.Slice, reflect.Map:
		default:
			return fmt.Errorf("just support ptr to struct|slice|map")
		}

		if wr.InputBody == nil {
			inputBody = reflect.New(bodyType).Elem().Interface()
		}

		if inputBody != nil {
			p.buildBody(wr.Consume, inputBody)
		}
	}

	// build output head & body
	{
		output := wr.Output
		if numOut == 2 {
			ot := t.Out(0)
			if ot.Kind() == reflect.Ptr {
				ot = ot.Elem()
			}

			if output == nil {
				output = reflect.New(ot).Elem().Interface()
			}
		}
		wr.RespWriter.AddRoute(wr.Method, path.Join(p.ws.RootPath(), wr.SubPath))
		b.Returns(http.StatusOK, "OK", output)
	}

	handler := func(req *restful.Request, resp *restful.Response) {
		var ret []reflect.Value

		if numIn == 4 {
			// with param & body
			param := reflect.New(paramType).Interface()
			body := reflect.New(bodyType).Interface()

			if err := ReadEntity(req, param, body, p.parameterCodec); err != nil {
				responsewriters.Error(err, resp.ResponseWriter, req.Request)
				return
			}

			// audit
			ae := request.AuditEventFrom(req.Request.Context())
			audit.LogRequestObject(ae, body, "")

			// TODO: use (w http.ResponseWriter, req *http.Request)
			ret = v.Call([]reflect.Value{
				reflect.ValueOf(resp.ResponseWriter),
				reflect.ValueOf(req.Request),
				reflect.ValueOf(param),
				reflect.ValueOf(body),
			})

		} else if numIn == 3 {
			// with param
			param := reflect.New(paramType).Interface()
			if err := ReadEntity(req, param, nil, p.parameterCodec); err != nil {
				responsewriters.Error(err, resp.ResponseWriter, req.Request)
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
			wr.RespWriter.RespWrite(resp, req.Request, toInterface(ret[0]), toError(ret[1]))
			return
		}

		if numOut == 1 {
			wr.RespWriter.RespWrite(resp, req.Request, nil, toError(ret[0]))
		}
	}

	handler = metrics.InstrumentRouteFunc(wr.Method,
		path.Join(p.ws.RootPath(), wr.SubPath),
		metrics.APIServerComponent,
		wr.Deprecated,
		handler,
	)

	b.To(handler)
	return nil
}

func (p *RouteBuilder) buildParam(param interface{}) *RouteBuilder {
	p.parameterCodec.RouteBuilderParameters(p.rb, param)
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

func toInterface(v reflect.Value) interface{} {
	if v.IsNil() {
		return nil
	}
	if v.CanInterface() {
		return v.Interface()
	}
	return nil
}

func toError(v reflect.Value) error {
	if v.CanInterface() {
		e, _ := v.Interface().(error)
		return e
	}
	return nil
}

func ScopeRegister(scope, description string) {
	ScopeCatalog[scope] = description
}

func SecurityScheme(ssoAddr string) *spec.SecurityScheme {
	return spec.OAuth2AccessToken(ssoAddr+"/o/oauth2/authorize", ssoAddr+"/o/oauth2/token")
}

func ResponseWriterRegister(w RespWriter) error {
	name := w.Name()
	if respWriterCatalog[name] != nil {
		return fmt.Errorf("ResponseWriterRegister %s exists", name)
	}

	klog.V(3).Infof("add resp writer %s", name)
	respWriterCatalog[name] = w
	return nil
}

func SecuritySchemeRegister(name string, s *spec.SecurityScheme) error {
	if securitySchemeCatalog[name] != nil {
		return fmt.Errorf("SecuritySchemeRegister %s exists", name)
	}

	for scope, desc := range ScopeCatalog {
		klog.Infof("scope %s %s", scope, desc)
		s.AddScope(scope, desc)
	}

	klog.V(3).Infof("add scheme %s", name)
	securitySchemeCatalog[name] = s
	return nil
}

func SchemeRegisterBasicAuth() error {
	return SecuritySchemeRegister(string(SecurityTypeBase), spec.BasicAuth())
}

func SchemeRegisterApiKey(fieldName, valueSource string) error {
	return SecuritySchemeRegister(string(SecurityTypeApiKey),
		spec.APIKeyAuth(fieldName, valueSource))
}

func SchemeRegisterOAuth2Implicit(authorizationURL string) error {
	return SecuritySchemeRegister(string(SecurityTypeImplicit),
		spec.OAuth2Implicit(authorizationURL))
}

func SchemeRegisterOAuth2Password(tokenURL string) error {
	return SecuritySchemeRegister(string(SecurityTypePassword),
		spec.OAuth2Password(tokenURL))
}
func SchemeRegisterOAuth2Application(tokenURL string) error {
	return SecuritySchemeRegister(string(SecurityTypeApplication),
		spec.OAuth2Application(tokenURL))
}

func SchemeRegisterOAuth2AccessToken(authorizationURL, tokenURL string) error {
	return SecuritySchemeRegister(string(SecurityTypeAccessCode),
		spec.OAuth2AccessToken(authorizationURL, tokenURL))
}

func SwaggerTagsRegister(tags ...spec.Tag) {
	swaggerTags = append(swaggerTags, tags...)
}

func SwaggerTagRegister(name, desc string) {
	for _, v := range swaggerTags {
		if v.Name == name {
			klog.Warningf("SwaggerTagRegister %s has been added", name)
			return
		}
	}

	swaggerTags = append(swaggerTags, spec.Tag{TagProps: spec.TagProps{
		Name:        name,
		Description: desc,
	}})
}

func InstallApiDocs(apiPath string, container *restful.Container, infoProps spec.InfoProps, securitySchemes []SchemeConfig) error {
	// register scheme to openapi
	for _, v := range securitySchemes {
		scheme, err := v.SecurityScheme()
		if err != nil {
			return err
		}

		if err := SecuritySchemeRegister(v.Name, scheme); err != nil {
			return err
		}
	}

	// apidocs
	wss := container.RegisteredWebServices()
	ws := restfulspec.NewOpenAPIService(restfulspec.Config{
		// you control what services are visible
		WebServices:                   wss,
		APIPath:                       apiPath,
		PostBuildSwaggerObjectHandler: genSwaggerHandler(wss, infoProps),
	})
	container.Add(ws)
	return nil
}

func genSwaggerHandler(wss []*restful.WebService, infoProps spec.InfoProps) func(*spec.Swagger) {
	return func(s *spec.Swagger) {
		for _, v := range respWriterCatalog {
			v.SwaggerHandler(s)
		}

		swaggerWithSecurityScheme(wss, infoProps, s)
	}
}

func swaggerWithSecurityScheme(wss []*restful.WebService, infoProps spec.InfoProps, s *spec.Swagger) {

	s.Info = &spec.Info{InfoProps: infoProps}
	s.Tags = swaggerTags
	s.SecurityDefinitions = securitySchemeCatalog

	if len(s.SecurityDefinitions) == 0 {
		return
	}

	// loop through all registerd web services
	for _, ws := range wss {
		for _, route := range ws.Routes() {
			// route metadata for a SecurityDefinition
			secdefn, ok := route.Metadata[SecurityDefinitionKey]
			if !ok {
				continue
			}

			scope, ok := secdefn.(string)
			if !ok {
				continue
			}

			operation, err := OperationFrom(s, route.Method, route.Path)
			if err != nil {
				panic(err)
			}

			// update the operation with security entry
			for k := range s.SecurityDefinitions {
				operation.SecuredWith(k, scope)
			}
		}
	}
}

type SchemeConfig struct {
	Name             string       `json:"name"`
	Type             SecurityType `json:"type" description:"base|apiKey|implicit|password|application|accessCode"`
	FieldName        string       `json:"fieldName" description:"used for apiKey"`
	ValueSource      string       `json:"valueSource" description:"used for apiKey, header|query|cookie"`
	AuthorizationURL string       `json:"authorizationURL" description:"used for OAuth2"`
	TokenURL         string       `json:"tokenURL" description:"used for OAuth2"`
}

func (p *SchemeConfig) SecurityScheme() (*spec.SecurityScheme, error) {
	if p.Name == "" {
		return nil, fmt.Errorf("name must be set")
	}
	switch p.Type {
	case SecurityTypeBase:
		return spec.BasicAuth(), nil
	case SecurityTypeApiKey:
		if p.FieldName == "" {
			return nil, fmt.Errorf("fieldName must be set for %s", p.Type)
		}
		if p.ValueSource == "" {
			return nil, fmt.Errorf("valueSource must be set for %s", p.Type)
		}
		return spec.APIKeyAuth(p.FieldName, p.ValueSource), nil
	case SecurityTypeImplicit:
		if p.AuthorizationURL == "" {
			return nil, fmt.Errorf("authorizationURL must be set for %s", p.Type)
		}
		return spec.OAuth2Implicit(p.AuthorizationURL), nil
	case SecurityTypePassword:
		if p.TokenURL == "" {
			return nil, fmt.Errorf("tokenURL must be set for %s", p.Type)
		}
		return spec.OAuth2Password(p.TokenURL), nil
	case SecurityTypeApplication:
		if p.TokenURL == "" {
			return nil, fmt.Errorf("tokenURL must be set for %s", p.Type)
		}
		return spec.OAuth2Application(p.TokenURL), nil
	case SecurityTypeAccessCode:
		if p.TokenURL == "" {
			return nil, fmt.Errorf("tokenURL must be set for %s", p.Type)
		}
		if p.AuthorizationURL == "" {
			return nil, fmt.Errorf("authorizationURL must be set for %s", p.Type)
		}
		return spec.OAuth2AccessToken(p.AuthorizationURL, p.TokenURL), nil
	default:
		return nil, fmt.Errorf("scheme.type %s is invalid, should be one of %s", p.Type,
			strings.Join([]string{
				string(SecurityTypeBase),
				string(SecurityTypeApiKey),
				string(SecurityTypeImplicit),
				string(SecurityTypePassword),
				string(SecurityTypeApplication),
				string(SecurityTypeAccessCode),
			}, ", "))
	}
}
