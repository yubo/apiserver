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
	"github.com/yubo/golib/runtime"
	"github.com/yubo/golib/scheme"
	"k8s.io/klog/v2"
)

var (
	ScopeCatalog          = map[string]string{}
	securitySchemeCatalog = map[string]*spec.SecurityScheme{}
	respWriterCatalog     = map[string]RespWriter{}
	swaggerTags           = []spec.Tag{}
	DefaultContentTypes   = []string{MIME_JSON}

	defaultWebServiceBuilder = NewWebServiceBudiler()
)

func NewWebServiceBudiler() *WebServiceBuilder {
	return &WebServiceBuilder{
		ScopeCatalog:          map[string]string{},
		securitySchemeCatalog: map[string]*spec.SecurityScheme{},
		respWriterCatalog:     map[string]RespWriter{},
		swaggerTags:           []spec.Tag{},
		DefaultContentTypes:   []string{MIME_JSON},
	}
}

func WsRouteBuild(opt *WsOption) {
	defaultWebServiceBuilder.Build(opt)
}

func ScopeRegister(scope, description string) {
	defaultWebServiceBuilder.ScopeRegister(scope, description)
}
func SecuritySchemeRegister(name string, s *spec.SecurityScheme) error {
	return defaultWebServiceBuilder.SecuritySchemeRegister(name, s)
}
func InstallApiDocs(apiPath string, container *restful.Container, infoProps spec.InfoProps, securitySchemes []SchemeConfig) error {
	return defaultWebServiceBuilder.InstallApiDocs(apiPath, container, infoProps, securitySchemes)
}

func ResponseWriterRegister(w RespWriter) error {
	return defaultWebServiceBuilder.ResponseWriterRegister(w)
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

type WebServiceBuilder struct {
	ScopeCatalog          map[string]string
	securitySchemeCatalog map[string]*spec.SecurityScheme
	respWriterCatalog     map[string]RespWriter
	swaggerTags           []spec.Tag
	DefaultContentTypes   []string
}

func (p *WebServiceBuilder) Build(opt *WsOption) {
	p.newBuilder(opt).build()
}

func (p *WebServiceBuilder) newBuilder(opts *WsOption) *webserviceBuilder {
	if err := opts.Validate(); err != nil {
		panic(err)
	}

	wb := &webserviceBuilder{
		WebServiceBuilder: p,
		WsOption:          opts,
		ws:                opts.Ws,
		parameterCodec:    ParameterCodec,
		container:         opts.GoRestfulContainer,
		serializer:        scheme.NegotiatedSerializer,
	}
	if wb.container != nil {
		wb.serializer = opts.GoRestfulContainer.Serializer()
	}

	return wb
}

func (p *WebServiceBuilder) ScopeRegister(scope, description string) {
	p.ScopeCatalog[scope] = description
}

func (p *WebServiceBuilder) SecuritySchemeRegister(name string, s *spec.SecurityScheme) error {
	if p.securitySchemeCatalog[name] != nil {
		return fmt.Errorf("SecuritySchemeRegister %s exists", name)
	}

	for scope, desc := range p.ScopeCatalog {
		klog.Infof("scope %s %s", scope, desc)
		s.AddScope(scope, desc)
	}

	klog.V(3).Infof("add scheme %s", name)
	p.securitySchemeCatalog[name] = s
	return nil
}

func (p *WebServiceBuilder) InstallApiDocs(apiPath string, container *restful.Container, infoProps spec.InfoProps, securitySchemes []SchemeConfig) error {
	// register scheme to openapi
	for _, v := range securitySchemes {
		scheme, err := v.SecurityScheme()
		if err != nil {
			return err
		}

		if err := p.SecuritySchemeRegister(v.Name, scheme); err != nil {
			return err
		}
	}

	// apidocs
	wss := container.RegisteredWebServices()
	ws := restfulspec.NewOpenAPIService(restfulspec.Config{
		// you control what services are visible
		WebServices:                   wss,
		APIPath:                       apiPath,
		PostBuildSwaggerObjectHandler: p.genSwaggerHandler(wss, infoProps),
	})
	container.Add(ws)
	return nil
}

func (p *WebServiceBuilder) SwaggerTagsRegister(tags ...spec.Tag) {
	p.swaggerTags = append(p.swaggerTags, tags...)
}

func (p *WebServiceBuilder) SwaggerTagRegister(name, desc string) {
	for _, v := range p.swaggerTags {
		if v.Name == name {
			klog.Warningf("SwaggerTagRegister %s has been added", name)
			return
		}
	}

	p.swaggerTags = append(p.swaggerTags, spec.Tag{TagProps: spec.TagProps{
		Name:        name,
		Description: desc,
	}})
}

func (p *WebServiceBuilder) ResponseWriterRegister(w RespWriter) error {
	name := w.Name()
	if p.respWriterCatalog[name] != nil {
		return fmt.Errorf("ResponseWriterRegister %s exists", name)
	}

	klog.V(3).Infof("add resp writer %s", name)
	p.respWriterCatalog[name] = w
	return nil
}

func (p *WebServiceBuilder) genSwaggerHandler(wss []*restful.WebService, infoProps spec.InfoProps) func(*spec.Swagger) {
	return func(s *spec.Swagger) {
		for _, v := range p.respWriterCatalog {
			v.SwaggerHandler(s)
		}

		p.swaggerWithSecurityScheme(wss, infoProps, s)
	}
}

func (p *WebServiceBuilder) swaggerWithSecurityScheme(wss []*restful.WebService, infoProps spec.InfoProps, s *spec.Swagger) {

	s.Info = &spec.Info{InfoProps: infoProps}
	s.Tags = p.swaggerTags
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

type webserviceBuilder struct {
	*WebServiceBuilder
	*WsOption
	ws             *restful.WebService
	container      GoRestfulContainer
	parameterCodec request.ParameterCodec
	serializer     runtime.NegotiatedSerializer
}

func (p *webserviceBuilder) build() {
	routes := p.Routes

	for i := range routes {
		rb := p.newRouteBuilder(routes[i])
		p.ws.Route(rb)
	}

	if p.container != nil {
		p.container.Add(p.ws)
	}
}

func (p *webserviceBuilder) newRouteBuilder(wr WsRoute) *restful.RouteBuilder {

	//func (p *webserviceBuilder) apply(opt *WsOption, wr *WsRoute) *restful.RouteBuilder {
	var rb *restful.RouteBuilder
	opt := p.WsOption

	wr.SubPath = opt.PrefixPath + wr.SubPath

	{
		// opt.Filter > opt.Filters > route.acl > route.filter > route.filters
		var filters []restful.FilterFunction
		if opt.Filter != nil {
			filters = append(filters, opt.Filter)
		}

		if len(opt.Filters) > 0 {
			filters = append(filters, opt.Filters...)
		}

		if wr.Acl != "" && opt.Acl != nil {
			var filter restful.FilterFunction
			var err error
			if filter, wr.Scope, err = opt.Acl(wr.Acl); err != nil {
				panic(err)
			}
			filters = append(filters, filter)
		}

		if len(wr.Filters) > 0 {
			filters = append(filters, wr.Filters...)
		}

		wr.Filters = filters
	}

	if wr.Acl != "" {
		wr.Desc += " acl(" + wr.Acl + ")"
	}

	if wr.Scope != "" {
		wr.Desc += " scope(" + wr.Scope + ")"
	}

	if wr.Tags == nil && opt.Tags != nil {
		wr.Tags = opt.Tags
	}

	if wr.RespWriter == nil {
		wr.RespWriter = opt.RespWriter
	}

	switch strings.ToUpper(wr.Method) {
	case "GET", "LIST":
		rb = p.ws.GET(wr.SubPath)
	case "POST", "CREATE":
		rb = p.ws.POST(wr.SubPath)
	case "PUT", "UPDATE":
		rb = p.ws.PUT(wr.SubPath)
	case "DELETE":
		rb = p.ws.DELETE(wr.SubPath)
	default:
		panic("unsupported method " + wr.Method)
	}

	if wr.Deprecated {
		rb.Deprecate()
	}

	if wr.Scope != "" {
		rb.Metadata(SecurityDefinitionKey, wr.Scope)
	}

	if wr.Consume != "" {
		rb.Consumes(wr.Consume)
	}

	if wr.Produce != "" {
		rb.Produces(wr.Produce)
	}

	if wr.Operation != "" {
		rb.Operation(wr.Operation)
	}

	if wr.Notes != "" {
		rb.Notes(wr.Notes)
	}

	for _, filter := range wr.Filters {
		rb.Filter(filter)
	}

	for _, out := range wr.ExtraOutput {
		rb.Returns(out.Code, out.Message, out.Model)
	}

	if err := p.registerHandle(rb, &wr); err != nil {
		panic(err)
	}

	rb.Doc(wr.Desc)
	rb.Metadata(restfulspec.KeyOpenAPITags, wr.Tags)

	return rb
}

func (p *webserviceBuilder) registerHandle(rb *restful.RouteBuilder, wr *WsRoute) error {
	if wr.Handle == nil {
		rb.To(noneHandle)
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
			p.parameterCodec.RouteBuilderParameters(rb, inputParam)
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
			p.buildBody(rb, wr.Consume, inputBody)
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
		rb.Returns(http.StatusOK, http.StatusText(http.StatusOK), output)
	}

	handler := func(req *restful.Request, resp *restful.Response) {
		var ret []reflect.Value

		if numIn == 4 {
			// with param & body
			param := reflect.New(paramType).Interface()
			body := reflect.New(bodyType).Interface()

			if err := readEntity(req, param, body, p.parameterCodec); err != nil {
				responsewriters.ErrorNegotiated(err, p.serializer, resp.ResponseWriter, req.Request)
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
			if err := readEntity(req, param, nil, p.parameterCodec); err != nil {
				responsewriters.ErrorNegotiated(err, p.serializer, resp.ResponseWriter, req.Request)
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
			wr.RespWriter.RespWrite(resp, req.Request, toInterface(ret[0]), toError(ret[1]), p.serializer)
			return
		}

		if numOut == 1 {
			wr.RespWriter.RespWrite(resp, req.Request, nil, toError(ret[0]), p.serializer)
		}
	}

	handler = metrics.InstrumentRouteFunc(wr.Method,
		path.Join(p.ws.RootPath(), wr.SubPath),
		metrics.APIServerComponent,
		wr.Deprecated,
		handler,
	)

	rb.To(handler)
	return nil
}

func (p *webserviceBuilder) buildBody(rb *restful.RouteBuilder, consume string, body interface{}) {
	rv := reflect.Indirect(reflect.ValueOf(body))
	rt := rv.Type()

	klog.V(10).Infof("buildbody %s", rt.Name())
	if consume == MIME_URL_ENCODED {
		err := urlencoded.RouteBuilderReads(rb, rv)
		if err != nil {
			panic(err)
		}
		return
	}

	rb.Reads(rv.Interface())
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

	Serializer() runtime.NegotiatedSerializer
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
		p.ParameterCodec = ParameterCodec
	}
	if p.RespWriter == nil {
		p.RespWriter = DefaultRespWriter
	}
	if p.GoRestfulContainer != nil {
		klog.Warningf("unable to get RestFulContainer, routebuild %s", p.Path)
	}
	return nil
}

type NonParam struct{}

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

func noneHandle(req *restful.Request, resp *restful.Response) {}

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

//func SecurityScheme(ssoAddr string) *spec.SecurityScheme {
//	return spec.OAuth2AccessToken(ssoAddr+"/o/oauth2/authorize", ssoAddr+"/o/oauth2/token")
//}

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

// dst: must be ptr
func readEntity(req *restful.Request, param, body interface{}, codec request.ParameterCodec) error {
	ctx := request.WithParam(req.Request.Context(), param)
	ctx = request.WithBody(ctx, body)
	req.Request = req.Request.WithContext(ctx)

	// param
	if err := codec.DecodeParameters(&request.Parameters{
		Header: req.Request.Header,
		Path:   req.PathParameters(),
		Query:  req.Request.URL.Query(),
	}, param); err != nil {
		return nil
	}
	if v, ok := param.(Validator); ok {
		if err := v.Validate(); err != nil {
			return err
		}
	}

	// TODO: use scheme.Codecs instead of restful.ReadEntity
	// body
	if body != nil {
		if err := req.ReadEntity(body); err != nil {
			klog.V(5).Infof("restful.ReadEntity() error %s", err)
			return err
		}
	}
	if v, ok := body.(Validator); ok {
		if err := v.Validate(); err != nil {
			return err
		}
	}

	return nil
}
