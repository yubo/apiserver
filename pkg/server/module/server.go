package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	rt "runtime"
	"time"

	"github.com/emicklei/go-restful/v3"
	"github.com/go-openapi/spec"
	"github.com/google/uuid"
	"github.com/yubo/apiserver/components/dbus"
	"github.com/yubo/apiserver/components/logs"
	"github.com/yubo/apiserver/components/version"
	"github.com/yubo/apiserver/pkg/authorization/authorizerfactory"
	"github.com/yubo/apiserver/pkg/filters"
	"github.com/yubo/apiserver/pkg/proc"
	v1 "github.com/yubo/apiserver/pkg/proc/api/v1"
	"github.com/yubo/apiserver/pkg/server"
	"github.com/yubo/apiserver/pkg/server/config"
	"github.com/yubo/apiserver/pkg/server/healthz"
	"github.com/yubo/apiserver/pkg/server/routes"
	"github.com/yubo/client-go/rest"
	"github.com/yubo/golib/configer"
	"github.com/yubo/golib/runtime"
	"github.com/yubo/golib/scheme"
	"github.com/yubo/golib/util/sets"
	utilwaitgroup "github.com/yubo/golib/util/waitgroup"
	"go.opentelemetry.io/otel"
	"k8s.io/klog/v2"
)

const (
	moduleName = "apiserver"
)

var (
	_module = &serverModule{name: moduleName}
	hookOps = []v1.HookOps{{
		Hook:        _module.init,
		Owner:       moduleName,
		HookNum:     v1.ACTION_START,
		Priority:    v1.PRI_SYS_INIT,
		SubPriority: v1.PRI_M_HTTP,
	}, {
		Hook:        _module.init2,
		Owner:       moduleName,
		HookNum:     v1.ACTION_START,
		Priority:    v1.PRI_SYS_INIT,
		SubPriority: v1.PRI_M_HTTP2,
	}, {
		Hook:        _module.start,
		Owner:       moduleName,
		HookNum:     v1.ACTION_START,
		Priority:    v1.PRI_SYS_START,
		SubPriority: v1.PRI_M_HTTP,
	}, {
		Hook:        _module.stop,
		Owner:       moduleName,
		HookNum:     v1.ACTION_STOP,
		Priority:    v1.PRI_SYS_START,
		SubPriority: v1.PRI_M_HTTP,
	}}
)

var _ server.APIServer = &serverModule{}

type serverModule struct {
	name   string
	config *config.Config
	server *server.Config

	ctx       context.Context
	cancel    context.CancelFunc
	stoppedCh chan struct{}
}

// init: no dep
func (p *serverModule) init(ctx context.Context) (err error) {
	p.ctx, p.cancel = context.WithCancel(ctx)

	cf := config.NewConfig()
	if err := proc.ReadConfig(p.name, cf); err != nil {
		return err
	}

	p.config = cf
	p.server = cf.NewServerConfig()

	if err := p.serverInit(); err != nil {
		return err
	}

	dbus.RegisterAPIServer(p)

	return nil
}

// init: dep authn, authz, audit
func (p *serverModule) init2(ctx context.Context) (err error) {
	if err := p.serverInit2(); err != nil {
		return err
	}

	return nil
}

func (p *serverModule) Address() string {
	return p.server.SecureServing.Listener.Addr().String()
}

func (p *serverModule) start(ctx context.Context) error {
	if err := p.Start(p.ctx.Done(), p.stoppedCh); err != nil {
		return err
	}

	p.Info()

	return nil
}

func (p *serverModule) stop(ctx context.Context) error {
	if p.cancel == nil {
		return nil
	}

	p.cancel()

	<-p.stoppedCh

	return nil
}

func (p *serverModule) Info() {
	if !klog.V(10).Enabled() {
		return
	}
	for _, path := range p.server.Handler.ListedPaths() {
		klog.V(1).Infof("apiserver path %s", path)
	}
}
func (p *serverModule) Config() *server.Config {
	return p.server
}

// Add a WebService to the Container. It will detect duplicate root paths and exit in that case.
func (p *serverModule) Add(service *restful.WebService) *restful.Container {
	return p.server.Handler.GoRestfulContainer.Add(service)
}

// Remove a WebService from the Container.
func (p *serverModule) Remove(service *restful.WebService) error {
	return p.server.Handler.GoRestfulContainer.Remove(service)
}

// Handle registers the handler for the given pattern.
// If a handler already exists for pattern, Handle panics.
func (p *serverModule) Handle(path string, handler http.Handler) {
	p.server.Handler.NonGoRestfulMux.Handle(path, handler)
}

// UnlistedHandle registers the handler for the given pattern, but doesn't list it.
// If a handler already exists for pattern, Handle panics.
func (p *serverModule) UnlistedHandle(path string, handler http.Handler) {
	p.server.Handler.NonGoRestfulMux.UnlistedHandle(path, handler)
}

// HandlePrefix is like Handle, but matches for anything under the path.  Like a standard golang trailing slash.
func (p *serverModule) HandlePrefix(path string, handler http.Handler) {
	p.server.Handler.NonGoRestfulMux.HandlePrefix(path, handler)
}

// UnlistedHandlePrefix is like UnlistedHandle, but matches for anything under the path.  Like a standard golang trailing slash.
func (p *serverModule) UnlistedHandlePrefix(path string, handler http.Handler) {
	p.server.Handler.NonGoRestfulMux.UnlistedHandlePrefix(path, handler)
}

// ListedPaths is an alphabetically sorted list of paths to be reported at /.
func (p *serverModule) ListedPaths() []string {
	return p.server.ListedPathProvider.ListedPaths()
}

func (p *serverModule) Serializer() runtime.NegotiatedSerializer {
	return p.server.Serializer
}

// Filter appends a container FilterFunction. These are called before dispatching
// a http.Request to a WebService from the container
func (p *serverModule) Filter(filter restful.FilterFunction) {
	p.server.Handler.GoRestfulContainer.Filter(filter)
}

func (p *serverModule) serverInit() error {
	if p == nil || p.server == nil || p.config == nil {
		return nil
	}

	if err := p.prepare(); err != nil {
		return err
	}

	if err := p.servingInit(); err != nil {
		return err
	}

	return nil
}

// serverInit2 call before start
func (p *serverModule) serverInit2() error {
	if p == nil || p.server == nil || p.config == nil {
		return nil
	}

	if err := p.authInit(); err != nil {
		return err
	}

	if err := p.handlerInit(); err != nil {
		return err
	}

	if err := p.installAPI(); err != nil {
		return err
	}

	return nil
}

func (p *serverModule) prepare() error {
	c := p.config
	s := p.server

	if err := c.GenericServerRunOptions.DefaultAdvertiseAddress(c.SecureServing); err != nil {
		return err
	}

	if err := c.SecureServing.MaybeDefaultWithSelfSignedCerts(
		c.GenericServerRunOptions.AdvertiseAddress.String(), c.AlternateDNS,
		[]net.IP{}); err != nil {
		return fmt.Errorf("error creating self-signed certificates: %v", err)
	}

	if len(c.GenericServerRunOptions.ExternalHost) == 0 {
		if len(c.GenericServerRunOptions.AdvertiseAddress) > 0 {
			c.GenericServerRunOptions.ExternalHost = c.GenericServerRunOptions.AdvertiseAddress.String()
		} else {
			if hostname, err := os.Hostname(); err == nil {
				c.GenericServerRunOptions.ExternalHost = hostname
			} else {
				return fmt.Errorf("error finding host name: %v", err)
			}
		}
		klog.Infof("external host was not specified, using %v", c.GenericServerRunOptions.ExternalHost)
	}

	p.stoppedCh = make(chan struct{})
	s.ApiServerID = proc.Name() + "-" + uuid.New().String()

	if s.BuildHandlerChainFunc == nil {
		s.BuildHandlerChainFunc = server.DefaultBuildHandlerChain
	}
	if s.HandlerChainWaitGroup == nil {
		s.HandlerChainWaitGroup = new(utilwaitgroup.SafeWaitGroup)
	}
	if s.RequestInfoResolver == nil {
		s.RequestInfoResolver = server.NewRequestInfoResolver(s)
	}
	// start of buildGenericConfig
	s.LongRunningFunc = filters.BasicLongRunningRequestCheck(
		sets.NewString("watch", "proxy"),
		sets.NewString("attache", "exec", "proxyp", "log", "rotforward"),
	)

	version := version.Get()
	s.Version = &version

	return nil
}

// servingInit initialize secureServing / insecureServing/ loopbackClientConfig
func (p *serverModule) servingInit() error {
	s := p.server
	c := p.config

	if err := c.InsecureServing.ApplyToWithLoopback(&s.InsecureServing, &s.LoopbackClientConfig); err != nil {
		return err
	}
	if err := c.SecureServing.ApplyToWithLoopback(&s.SecureServing, &s.LoopbackClientConfig); err != nil {
		return err
	}

	s.LoopbackClientConfig.ContentConfig = rest.ContentConfig{
		NegotiatedSerializer: scheme.Codecs,
	}
	// Disable compression for self-communication, since we are going to be
	// on a fast local network
	s.LoopbackClientConfig.DisableCompression = true

	return nil
}

func (p *serverModule) authInit() error {
	c := p.server

	c.TracerProvider = otel.GetTracerProvider()

	if audit, _ := dbus.GetAuditor(); audit != nil {
		c.AuditBackend = audit.Backend()
		c.AuditPolicyRuleEvaluator = audit.AuditPolicyRuleEvaluator()
	}

	if rhc, _ := dbus.GetRequestHeaderConfig(); rhc != nil {
		c.RequestHeaderConfig = rhc
	}

	if authz, _ := dbus.GetAuthorizationInfo(); authz != nil {
		c.Authorization = authz
	} else {
		c.Authorization = &server.AuthorizationInfo{
			Authorizer: authorizerfactory.NewAlwaysAllowAuthorizer(),
			Modes:      sets.NewString("AlwaysAllow"),
		}
	}
	klog.V(3).InfoS("Authorizer", "modes", c.Authorization.Modes)

	if authn, _ := dbus.GetAuthenticationInfo(); authn != nil {
		c.Authentication = authn
	} else {
		c.Authentication = &server.AuthenticationInfo{}
	}

	// ApplyAuthorization will conditionally modify the authentication options based on the authorization options
	// authorization ModeAlwaysAllow cannot be combined with AnonymousAuth.
	// in such a case the AnonymousAuth is stomped to false and you get a message
	if c.Authorization != nil && c.Authentication != nil &&
		c.Authentication.Anonymous &&
		c.Authorization.Modes.Has("AlwaysAllow") {
		return fmt.Errorf("AnonymousAuth is not allowed with the AlwaysAllow authorizer. Resetting AnonymousAuth to false. You should use a different authorizer")
	}

	server.AuthorizeClientBearerToken(c.LoopbackClientConfig, c.Authentication, c.Authorization)

	return nil
}

func (p *serverModule) handlerInit() error {
	s := p.server
	handlerChainBuilder := func(handler http.Handler) http.Handler {
		return s.BuildHandlerChainFunc(handler, s)
	}
	apiServerHandler := server.NewAPIServerHandler("apiserver", s.Serializer, handlerChainBuilder, nil)
	s.Handler = apiServerHandler
	s.ListedPathProvider = routes.ListedPathProviders{apiServerHandler}

	return nil
}

func (p *serverModule) installAPI() error {
	s := p.server
	c := p.config

	if c.EnableIndex {
		routes.Index{}.Install(s.ListedPathProvider, s.Handler.NonGoRestfulMux)
	}

	if c.EnableProfiling {
		routes.Profiling{}.Install(s.Handler.NonGoRestfulMux)
		if c.EnableContentionProfiling {
			rt.SetBlockProfileRate(1)
		}
		// so far, only logging related endpoints are considered valid to add for these debug flags.
		routes.DebugFlags{}.Install(s.Handler.NonGoRestfulMux, "v", routes.StringFlagPutHandler(logs.GlogSetter))

	}

	if c.EnableMetrics {
		if c.EnableProfiling {
			routes.MetricsWithReset{}.Install(s.Handler.NonGoRestfulMux)
		} else {
			routes.DefaultMetrics{}.Install(s.Handler.NonGoRestfulMux)
		}

	}

	routes.Version{Version: s.Version}.Install(s.Handler.GoRestfulContainer)

	if c.EnableExpvar {
		routes.Expvar{}.Install(s.Handler.NonGoRestfulMux)
	}

	if c.EnableOpenAPI {
		routes.Swagger{}.Install(s.Handler.NonGoRestfulMux, server.APIDocsPath)
	}

	if c.EnableHealthz {
		healthz.InstallHandler(s.Handler.NonGoRestfulMux)
	}

	return nil
}

func (p *serverModule) Start(stopCh <-chan struct{}, done chan struct{}) error {
	s := p.server

	if s.EnableOpenAPI {
		err := routes.OpenAPI{}.Install(server.APIDocsPath,
			p.server.Handler.GoRestfulContainer,
			spec.InfoProps{
				Description: proc.Description(),
				Title:       proc.Name(),
				Contact:     proc.Contact(),
				License:     proc.License(),
				Version:     s.Version.String(),
			},
			s.SecuritySchemes,
		)
		if err != nil {
			return err
		}
	}

	delayedStopCh := make(chan struct{})

	// close socket after delayed stopCh
	if s.SecureServing != nil {
		stoppedCh, err := s.SecureServing.Serve(s.Handler, s.RequestTimeout, delayedStopCh)
		if err != nil {
			return err
		}
		s.HandlerChainWaitGroup.Add(1)
		go func() {
			<-stoppedCh
			s.HandlerChainWaitGroup.Done()
		}()
	}
	if s.InsecureServing != nil {
		stoppedCh, err := s.InsecureServing.Serve(s.Handler, s.RequestTimeout, delayedStopCh)
		if err != nil {
			return err
		}
		s.HandlerChainWaitGroup.Add(1)
		go func() {
			<-stoppedCh
			s.HandlerChainWaitGroup.Done()
		}()
	}

	go func() {
		<-stopCh
		time.Sleep(s.ShutdownDelayDuration)
		close(delayedStopCh)
	}()

	go func() {
		<-stopCh

		// wait for the delayed stopCh before closing the handler chain (it rejects everything after Wait has been called).
		<-delayedStopCh

		// Wait for all requests to finish, which are bounded by the RequestTimeout variable.
		s.HandlerChainWaitGroup.Wait()

		close(done)
	}()
	return nil
}

func RegisterHooks() {
	proc.RegisterHooks(hookOps)
}

func RegisterFlags() {
	proc.AddConfig(moduleName, config.NewConfig(), proc.WithConfigGroup("APIServer"))
}

func Register() {
	RegisterHooks()
	RegisterFlags()
}

func WithoutTLS() proc.ProcessOption {
	return proc.WithConfigOptions(
		configer.WithDefaultYaml("apiserver", `
secureServing:
  enabled: false
insecureServing:
  enabled: true`),
	)
}
