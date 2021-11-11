package server

import (
	"fmt"
	"net"
	"net/http"
	"os"
	rt "runtime"
	"time"

	"github.com/emicklei/go-restful"
	"github.com/go-openapi/spec"
	"github.com/google/uuid"
	"github.com/yubo/apiserver/pkg/authorization/authorizerfactory"
	"github.com/yubo/apiserver/pkg/filters"
	"github.com/yubo/apiserver/pkg/options"
	"github.com/yubo/apiserver/pkg/rest"
	"github.com/yubo/apiserver/pkg/server"
	"github.com/yubo/apiserver/pkg/server/config"
	"github.com/yubo/apiserver/pkg/server/routes"
	"github.com/yubo/apiserver/pkg/version"
	"github.com/yubo/golib/logs"
	"github.com/yubo/golib/proc"
	"github.com/yubo/golib/scheme"
	"github.com/yubo/golib/util/sets"
	utilwaitgroup "github.com/yubo/golib/util/waitgroup"
	"k8s.io/klog/v2"
)

func (p *module) Config() *server.Config {
	return p.config
}

// same as http.Handle()
func (p *module) Handle(pattern string, handler http.Handler) {
	p.config.Handler.GoRestfulContainer.Handle(pattern, handler)
}
func (p *module) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	p.Handle(pattern, http.HandlerFunc(handler))
}
func (p *module) UnlistedHandle(pattern string, handler http.Handler) {
	p.config.Handler.NonGoRestfulMux.UnlistedHandle(pattern, handler)
}
func (p *module) UnlistedHandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	p.UnlistedHandle(pattern, http.HandlerFunc(handler))
}
func (p *module) Add(service *restful.WebService) *restful.Container {
	return p.config.Handler.GoRestfulContainer.Add(service)
}
func (p *module) Filter(filter restful.FilterFunction) {
	p.config.Handler.GoRestfulContainer.Filter(filter)
}

func (p *module) serverInit(c *config.Config) (err error) {
	if p == nil || p.config == nil || c == nil {
		return nil
	}

	if err := p.prepare(c); err != nil {
		return err
	}

	if err := p.servingInit(c); err != nil {
		return err
	}

	if err := p.authInit(c); err != nil {
		return err
	}

	if err := p.handlerInit(); err != nil {
		return err
	}

	if err := p.installAPI(c); err != nil {
		return err
	}

	return nil
}

func (p *module) prepare(c *config.Config) error {
	s := p.config

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
func (p *module) servingInit(c *config.Config) error {
	s := p.config

	if err := c.InsecureServing.ApplyToWithLoopback(&s.InsecureServing, &s.LoopbackClientConfig); err != nil {
		return err
	}
	if err := c.SecureServing.ApplyToWithLoopback(&s.SecureServing, &s.LoopbackClientConfig); err != nil {
		return err
	}

	klog.Infof("%+v", c.InsecureServing)
	klog.Infof("%+v", c.SecureServing)
	klog.Infof("%+v", s.LoopbackClientConfig)

	s.LoopbackClientConfig.ContentConfig = rest.ContentConfig{
		NegotiatedSerializer: scheme.Codecs,
	}
	// Disable compression for self-communication, since we are going to be
	// on a fast local network
	s.LoopbackClientConfig.DisableCompression = true

	return nil
}

func (p *module) authInit(c *config.Config) error {
	s := p.config

	// Deprecated
	if session, ok := options.SessionManagerFrom(p.ctx); ok {
		s.Session = session
	}

	if audit, ok := options.AuditFrom(p.ctx); ok {
		s.AuditBackend = audit.Backend()
		s.AuditPolicyChecker = audit.Checker()
	}

	if authz, ok := options.AuthzFrom(p.ctx); ok {
		s.Authorization = authz
	} else {
		s.Authorization = &server.AuthorizationInfo{
			Authorizer: authorizerfactory.NewAlwaysAllowAuthorizer(),
			Modes:      sets.NewString("AlwaysAllow"),
		}
	}

	if authn, ok := options.AuthnFrom(p.ctx); ok {
		s.Authentication = authn
	} else {
		s.Authentication = &server.AuthenticationInfo{}
	}

	// ApplyAuthorization will conditionally modify the authentication options based on the authorization options
	// authorization ModeAlwaysAllow cannot be combined with AnonymousAuth.
	// in such a case the AnonymousAuth is stomped to false and you get a message
	if s.Authorization != nil && s.Authentication != nil &&
		s.Authentication.Anonymous &&
		s.Authorization.Modes.Has("AlwaysAllow") {
		return fmt.Errorf("AnonymousAuth is not allowed with the AlwaysAllow authorizer. Resetting AnonymousAuth to false. You should use a different authorizer")
	}

	server.AuthorizeClientBearerToken(s.LoopbackClientConfig, s.Authentication, s.Authorization)

	return nil
}

func (p *module) handlerInit() error {
	s := p.config
	handlerChainBuilder := func(handler http.Handler) http.Handler {
		return s.BuildHandlerChainFunc(handler, s)
	}
	apiServerHandler := server.NewAPIServerHandler("apiserver", s.Serializer, handlerChainBuilder, nil)
	s.Handler = apiServerHandler
	s.ListedPathProvider = apiServerHandler

	return nil
}

func (p *module) installAPI(c *config.Config) error {
	s := p.config

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

	return nil
}

func (p *module) Start(stopCh <-chan struct{}, done chan struct{}) error {
	s := p.config

	rest.InstallApiDocs(
		p.config.Handler.GoRestfulContainer,
		spec.InfoProps{Title: proc.Name()},
		APIPath,
	)

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
