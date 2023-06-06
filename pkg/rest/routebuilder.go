package rest

import (
	"net/http"
	"path"
	"reflect"
	"strings"

	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	"github.com/emicklei/go-restful/v3"
	"github.com/go-openapi/spec"
	"github.com/yubo/apiserver/pkg/metrics"
	"github.com/yubo/apiserver/pkg/rest/urlencoded"
	"github.com/yubo/apiserver/pkg/scheme"
	"github.com/yubo/golib/runtime"
	"github.com/yubo/golib/util"
	"github.com/yubo/golib/util/errors"
	"k8s.io/klog/v2"
)

var (
	defaultContentTypes = []string{MIME_JSON}

	defaultWebServiceBuilder = NewWebServiceBudiler()
)

func NewWebServiceBudiler() *WebServiceBuilder {
	return &WebServiceBuilder{
		ScopeCatalog:          map[string]string{},
		securitySchemeCatalog: map[string]*spec.SecurityScheme{},
		respWriterCatalog:     map[string]RespWriter{},
		swaggerTags:           []spec.Tag{},
	}
}

func WsRouteBuild(opt *WsOption) {
	defaultWebServiceBuilder.Build(opt)
}

func SetDefaultAclManager(m AclManager) {
	defaultWebServiceBuilder.WithAclManager(m)
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

func SwaggerTagsRegister(tags ...spec.Tag) {
	defaultWebServiceBuilder.SwaggerTagsRegister(tags...)
}
func SwaggerTagRegister(name, desc string) {
	defaultWebServiceBuilder.SwaggerTagRegister(name, desc)
}
func ResponseWriterRegister(w RespWriter) error {
	return defaultWebServiceBuilder.ResponseWriterRegister(w)
}

func SchemeRegisterBasicAuth() error {
	return SecuritySchemeRegister(string(SecurityTypeBase), spec.BasicAuth())
}

func SchemeRegisterApiKey(fieldName, valueSource string) error {
	return SecuritySchemeRegister(string(SecurityTypeAPIKey),
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
	AclManager            AclManager
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
		//parameterCodec:    opts.ParameterCodec,
		container:  opts.GoRestfulContainer,
		serializer: scheme.NegotiatedSerializer,
	}
	if wb.container != nil {
		wb.serializer = opts.GoRestfulContainer.Serializer()
	}

	return wb
}

func (p *WebServiceBuilder) ScopeRegister(scope, description string) {
	p.ScopeCatalog[scope] = description
}

func (p *WebServiceBuilder) WithAclManager(m AclManager) {
	p.AclManager = m
}

func (p *WebServiceBuilder) SecuritySchemeRegister(name string, s *spec.SecurityScheme) error {
	if p.securitySchemeCatalog[name] != nil {
		return errors.Errorf("SecuritySchemeRegister %s exists", name)
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
		return errors.Errorf("ResponseWriterRegister %s exists", name)
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
	s.SecurityDefinitions = p.securitySchemeCatalog

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
	ws        *restful.WebService
	container GoRestfulContainer
	//parameterCodec api.ParameterCodec
	serializer runtime.NegotiatedSerializer
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

		if wr.Acl != "" && opt.AclManager != nil {
			acl, err := opt.AclManager.Get(wr.Acl)
			if err != nil {
				panic(err)
			}
			filters = append(filters, acl.Filter)
			wr.Scope = acl.Scope
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
		klog.FatalfDepth(4, "register %s unsupported method %s", path.Join(p.ws.RootPath(), wr.SubPath), wr.Method)
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
	} else {
		rb.Operation(util.Name(wr.Handle))
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
		klog.FatalfDepth(4, "register %s err %s", path.Join(p.ws.RootPath(), wr.SubPath), err)
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

	rh, err := NewRouteHandle(wr.Handle, p.serializer, p.RespWriter)
	if err != nil {
		return errors.Wrapf(err, "new route handle")
	}

	// build input param
	inputParam := wr.InputParam
	if inputParam == nil {
		inputParam = newInterface(rh.param)
	}
	if inputParam != nil {
		scheme.ParameterCodec.RouteBuilderParameters(rb, inputParam)
	}

	// build intput body
	inputBody := wr.InputBody
	if inputBody == nil {
		inputBody = newInterface(rh.body)
	}
	if inputBody != nil {
		p.buildBody(rb, wr.Consume, inputBody)
	}

	// build output head & body
	output := wr.Output
	if output == nil {
		output = newInterface(rh.out)
	}
	rb.Returns(http.StatusOK, http.StatusText(http.StatusOK), output)

	wr.RespWriter.AddRoute(wr.Method, path.Join(p.ws.RootPath(), wr.SubPath))

	handler := rh.Handler()

	handler = metrics.InstrumentRouteFunc(wr.Method,
		path.Join(p.ws.RootPath(), wr.SubPath),
		metrics.APIServerComponent,
		wr.Deprecated,
		handler,
	)

	klog.V(3).InfoS("route register", "method", wr.Method, "path", p.ws.RootPath()+wr.SubPath, "handle", wr.Handle)
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

type AclManager interface {
	Get(name string) (*Acl, error)
}

type Acl struct {
	Filter restful.FilterFunction
	Scope  string
}

// sys.Filters > opt.Filter > opt.Filters > route.acl > route.filter > route.filters
type WsOption struct {
	Ws                 *restful.WebService
	Path               string
	AclManager         AclManager
	Filter             restful.FilterFunction
	Filters            []restful.FilterFunction
	Produces           []string
	Consumes           []string
	PrefixPath         string
	Tags               []string
	Routes             []WsRoute
	RespWriter         RespWriter
	GoRestfulContainer GoRestfulContainer
	//ParameterCodec     api.ParameterCodec
}

func (p *WsOption) Validate() error {
	if p.Ws == nil {
		p.Ws = &restful.WebService{}
	}
	if p.Path != "" {
		p.Ws = p.Ws.Path(p.Path)
	}
	if p.AclManager == nil {
		p.AclManager = defaultWebServiceBuilder.AclManager
	}
	if p.Ws.RootPath() == "/" {
		klog.Warningf("rootpath is set to /, which may overwrite the existing route")
	}
	if len(p.Produces) > 0 {
		p.Ws.Produces(p.Produces...)
	} else {
		p.Ws.Produces(defaultContentTypes...)
	}
	if len(p.Consumes) > 0 {
		p.Ws.Consumes(p.Consumes...)
	} else {
		p.Ws.Consumes(defaultContentTypes...)
	}
	//if p.ParameterCodec == nil {
	//	p.ParameterCodec = scheme.ParameterCodec
	//}
	if p.RespWriter == nil {
		p.RespWriter = DefaultRespWriter
	}
	if p.GoRestfulContainer == nil {
		klog.Warningf("unable to get RestFulContainer, routebuild %s", p.Path)
	}
	return nil
}

type WsRoute struct {
	Acl     string // access name
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
	if v.CanInterface() {
		return v.Interface()
	}
	klog.Errorf("can't interface typeof %s", v.Kind())
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
	Type             SecurityType `json:"type" description:"base|bearer|token|implicit|password|application|accessCode"`
	FieldName        string       `json:"fieldName" description:"used for token"`
	ValueSource      string       `json:"valueSource" description:"used for token, header|query|cookie"`
	AuthorizationURL string       `json:"authorizationURL" description:"used for OAuth2"`
	TokenURL         string       `json:"tokenURL" description:"used for OAuth2"`
}

func (p *SchemeConfig) SecurityScheme() (*spec.SecurityScheme, error) {
	if p.Name == "" {
		return nil, errors.New("name must be set")
	}
	switch strings.ToLower(string(p.Type)) {
	case string(SecurityTypeBase):
		return spec.BasicAuth(), nil
	case string(SecurityTypeBearer):
		return spec.APIKeyAuth("Authorization", "header"), nil
	case string(SecurityTypeAPIKey):
		if p.FieldName == "" {
			return nil, errors.Errorf("fieldName must be set for %s", p.Type)
		}
		if p.ValueSource == "" {
			return nil, errors.Errorf("valueSource must be set for %s", p.Type)
		}
		return spec.APIKeyAuth(p.FieldName, p.ValueSource), nil
	case string(SecurityTypeImplicit):
		if p.AuthorizationURL == "" {
			return nil, errors.Errorf("authorizationURL must be set for %s", p.Type)
		}
		return spec.OAuth2Implicit(p.AuthorizationURL), nil
	case string(SecurityTypePassword):
		if p.TokenURL == "" {
			return nil, errors.Errorf("tokenURL must be set for %s", p.Type)
		}
		return spec.OAuth2Password(p.TokenURL), nil
	case string(SecurityTypeApplication):
		if p.TokenURL == "" {
			return nil, errors.Errorf("tokenURL must be set for %s", p.Type)
		}
		return spec.OAuth2Application(p.TokenURL), nil
	case string(SecurityTypeAccessCode):
		if p.TokenURL == "" {
			return nil, errors.Errorf("tokenURL must be set for %s", p.Type)
		}
		if p.AuthorizationURL == "" {
			return nil, errors.Errorf("authorizationURL must be set for %s", p.Type)
		}
		return spec.OAuth2AccessToken(p.AuthorizationURL, p.TokenURL), nil
	default:
		return nil, errors.Errorf("scheme.type %s is invalid, should be one of %s", p.Type,
			strings.Join([]string{
				string(SecurityTypeBase),
				string(SecurityTypeBearer),
				string(SecurityTypeAPIKey),
				string(SecurityTypeImplicit),
				string(SecurityTypePassword),
				string(SecurityTypeApplication),
				string(SecurityTypeAccessCode),
			}, ", "))
	}
}
