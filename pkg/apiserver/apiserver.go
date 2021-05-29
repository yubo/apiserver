/*
Copyright 2014 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package app does all of the work necessary to create a Kubernetes
// APIServer by binding together the API, master and APIServer infrastructure.
// It can be configured and called directly or via the hyperkube framework.
package apiserver

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	rt "runtime"
	"sort"
	"strconv"
	"time"

	"github.com/emicklei/go-restful"
	"github.com/google/uuid"
	"github.com/yubo/apiserver/pkg/apiserver/mux"
	"github.com/yubo/apiserver/pkg/filters"
	"github.com/yubo/apiserver/pkg/options"
	apirequest "github.com/yubo/apiserver/pkg/request"
	"github.com/yubo/apiserver/pkg/responsewriters"
	"github.com/yubo/apiserver/pkg/session"
	"github.com/yubo/golib/proc"
	apierrors "github.com/yubo/golib/staging/api/errors"
	utilsnet "github.com/yubo/golib/staging/util/net"
	utilruntime "github.com/yubo/golib/staging/util/runtime"
	"github.com/yubo/golib/staging/util/sets"
	utilwaitgroup "github.com/yubo/golib/staging/util/waitgroup"
	"k8s.io/klog/v2"
)

const (
	defaultKeepAlivePeriod = 3 * time.Minute
)

// same as http.Handle()
func (p *apiserver) Handle(pattern string, handler http.Handler) {
	p.server.Handler.GoRestfulContainer.Handle(pattern, handler)
}
func (p *apiserver) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	p.Handle(pattern, http.HandlerFunc(handler))
}
func (p *apiserver) UnlistedHandle(pattern string, handler http.Handler) {
	p.server.Handler.NonGoRestfulMux.UnlistedHandle(pattern, handler)
}
func (p *apiserver) UnlistedHandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	p.UnlistedHandle(pattern, http.HandlerFunc(handler))
}
func (p *apiserver) Add(service *restful.WebService) *restful.Container {
	return p.server.Handler.GoRestfulContainer.Add(service)
}
func (p *apiserver) Filter(filter restful.FilterFunction) {
	p.server.Handler.GoRestfulContainer.Filter(filter)
}

func (p *apiserver) serverInit() error {
	cf := p.config
	c := NewServer(p.ctx)
	c.CorsAllowedOriginList = cf.CorsAllowedOriginList
	c.HSTSDirectives = cf.HSTSDirectives
	c.ExternalAddress = cf.ExternalHost
	c.MaxRequestsInFlight = cf.MaxRequestsInFlight
	c.MaxMutatingRequestsInFlight = cf.MaxMutatingRequestsInFlight
	c.RequestTimeout = cf.RequestTimeout
	c.MinRequestTimeout = cf.MinRequestTimeout
	c.ShutdownDelayDuration = cf.ShutdownDelayDuration
	c.MaxRequestBodyBytes = cf.MaxRequestBodyBytes
	c.PublicAddress = cf.BindAddress

	if cf.Listener == nil {
		var err error
		addr := net.JoinHostPort(cf.BindAddress.String(), strconv.Itoa(cf.BindPort))

		cf.Listener, cf.BindPort, err = createListener(cf.BindNetwork, addr, net.ListenConfig{})
		if err != nil {
			return fmt.Errorf("failed to create listener: %v", err)
		}
	} else {
		if _, ok := cf.Listener.Addr().(*net.TCPAddr); !ok {
			return fmt.Errorf("failed to parse ip and port from listener")
		}
		cf.BindPort = cf.Listener.Addr().(*net.TCPAddr).Port
		cf.BindAddress = cf.Listener.Addr().(*net.TCPAddr).IP
	}

	c.Listener = cf.Listener

	if err := c.Validate(); err != nil {
		return err
	}

	if err := c.init(p.ctx); err != nil {
		return err
	}

	p.server = c
	p.stoppedCh = make(chan struct{})

	return nil
}

// Server is a structure used to configure a GenericAPIServer.
// Its members are sorted roughly in order of importance for composers.
type Server struct {
	// SecureServing is required to serve https
	//SecureServing *SecureServingInfo

	// Listener is the secure server network listener.
	Listener net.Listener

	// LoopbackClientConfig is a config for a privileged loopback connection to the API server
	// This is required for proper functioning of the PostStartHooks on a GenericAPIServer
	// TODO: move into SecureServing(WithLoopback) as soon as insecure serving is gone
	//LoopbackClientConfig *restclient.Config

	// EgressSelector provides a lookup mechanism for dialing outbound connections.
	// It does so based on a EgressSelectorConfiguration which was read at startup.
	// EgressSelector *egressselector.EgressSelector

	CorsAllowedOriginList []string
	HSTSDirectives        []string
	// FlowControl, if not nil, gives priority and fairness to request handling
	// FlowControl utilflowcontrol.Interface

	EnableIndex     bool
	EnableProfiling bool
	// EnableDiscovery bool
	// Requires generic profiling enabled
	EnableContentionProfiling bool
	EnableMetrics             bool

	//DisabledPostStartHooks sets.String
	// done values in this values for this map are ignored.
	//PostStartHooks map[string]PostStartHookConfigEntry

	// ExternalAddress is the host name to use for external (public internet) facing URLs (e.g. Swagger)
	// Will default to a value based on secure serving info and available ipv4 IPs.
	ExternalAddress string

	//===========================================================================
	// Fields you probably don't care about changing
	//===========================================================================

	// BuildHandlerChainFunc allows you to build custom handler chains by decorating the apiHandler.
	BuildHandlerChainFunc func(ctx context.Context, apiHandler http.Handler, c *Server) (secure http.Handler)
	// HandlerChainWaitGroup allows you to wait for all chain handlers exit after the server shutdown.
	HandlerChainWaitGroup *utilwaitgroup.SafeWaitGroup
	// RequestInfoResolver is used to assign attributes (used by admission and authorization) based on a request URL.
	// Use-cases that are like kubelets may need to customize this.
	RequestInfoResolver apirequest.RequestInfoResolver

	// RESTOptionsGetter is used to construct RESTStorage types via the generic registry.
	// RESTOptionsGetter genericregistry.RESTOptionsGetter

	// If specified, all requests except those which match the LongRunningFunc predicate will timeout
	// after this duration.
	RequestTimeout time.Duration
	// If specified, long running requests such as watch will be allocated a random timeout between this value, and
	// twice this value.  Note that it is up to the request handlers to ignore or honor this timeout. In seconds.
	MinRequestTimeout int

	ShutdownDelayDuration time.Duration

	// The limit on the total size increase all "copy" operations in a json
	// patch may cause.
	// This affects all places that applies json patch in the binary.
	// JSONPatchMaxCopyBytes int64
	// The limit on the request size that would be accepted and decoded in a write request
	// 0 means no limit.
	MaxRequestBodyBytes int64
	// MaxRequestsInFlight is the maximum number of parallel non-long-running requests. Every further
	// request has to wait. Applies only to non-mutating requests.
	MaxRequestsInFlight int
	// MaxMutatingRequestsInFlight is the maximum number of parallel mutating requests. Every further
	// request has to wait.
	MaxMutatingRequestsInFlight int
	// Predicate which is true for paths of long-running http requests
	LongRunningFunc apirequest.LongRunningRequestCheck

	// GoawayChance is the probability that send a GOAWAY to HTTP/2 clients. When client received
	// GOAWAY, the in-flight requests will not be affected and new requests will use
	// a new TCP connection to triggering re-balancing to another server behind the load balance.
	// Default to 0, means never send GOAWAY. Max is 0.02 to prevent break the apiserver.
	GoawayChance float64

	//===========================================================================
	// values below here are targets for removal
	//===========================================================================

	// PublicAddress is the IP address where members of the cluster (kubelet,
	// kube-proxy, services, etc.) can reach the GenericAPIServer.
	// If nil or 0.0.0.0, the host's default interface will be used.
	PublicAddress net.IP

	// EquivalentResourceRegistry provides information about resources equivalent to a given resource,
	// and the kind associated with a given resource. As resources are installed, they are registered here.
	// EquivalentResourceRegistry runtime.EquivalentResourceRegistry

	// APIServerID is the ID of this API server
	APIServerID string

	// ############ from GenericAPIServer
	// minRequestTimeout is how short the request timeout can be.  This is used to build the RESTHandler
	minRequestTimeout time.Duration

	// ShutdownTimeout is the timeout used for server shutdown. This specifies the timeout before server
	// gracefully shutdown returns.
	ShutdownTimeout time.Duration

	// "Outputs"
	// Handler holds the handlers being used by this API server
	Handler *APIServerHandler

	// The limit on the request body size that would be accepted and decoded in a write request.
	// 0 means no limit.
	maxRequestBodyBytes int64
}

// NewConfig returns a Config struct with the default values

func NewServer(ctx context.Context) *Server {
	return &Server{
		//Serializer:                  serializer.NewCodecFactory(Scheme),
		BuildHandlerChainFunc:       DefaultBuildHandlerChain,
		HandlerChainWaitGroup:       new(utilwaitgroup.SafeWaitGroup),
		EnableIndex:                 true,
		EnableProfiling:             true,
		EnableMetrics:               true,
		MaxRequestsInFlight:         400,
		MaxMutatingRequestsInFlight: 200,
		RequestTimeout:              time.Duration(60) * time.Second,
		MinRequestTimeout:           1800,
		ShutdownDelayDuration:       time.Duration(0),
		MaxRequestBodyBytes:         int64(3 * 1024 * 1024),
		LongRunningFunc:             filters.BasicLongRunningRequestCheck(sets.NewString("watch"), sets.NewString()),
		APIServerID:                 proc.NameFrom(ctx) + "-" + uuid.New().String(),
	}
}

func (c *Server) Validate() error {
	if len(c.ExternalAddress) == 0 && c.PublicAddress != nil {
		c.ExternalAddress = c.PublicAddress.String()
	}

	// if there is no port, and we listen on one securely, use that one
	if _, _, err := net.SplitHostPort(c.ExternalAddress); err != nil {
		if c.Listener == nil {
			return fmt.Errorf("cannot derive external address port without listening on a secure port.")
		}
		_, port, err := c.HostPort()
		if err != nil {
			return fmt.Errorf("cannot derive external address from the secure port: %v", err)
		}
		c.ExternalAddress = net.JoinHostPort(c.ExternalAddress, strconv.Itoa(port))
	}

	if c.RequestInfoResolver == nil {
		c.RequestInfoResolver = NewRequestInfoResolver(c)
	}

	return nil
}

// New creates a new server which logically combines the handling chain with the passed server.
// name is used to differentiate for logging. The handler chain in particular can be difficult as it starts delegating.
// delegationTarget may not be nil.
func (c *Server) init(ctx context.Context) error {
	handlerChainBuilder := func(ctx context.Context, handler http.Handler) http.Handler {
		return c.BuildHandlerChainFunc(ctx, handler, c)
	}

	apiServerHandler := NewAPIServerHandler(ctx, handlerChainBuilder)

	c.minRequestTimeout = time.Duration(c.MinRequestTimeout) * time.Second
	c.ShutdownTimeout = c.RequestTimeout
	c.Handler = apiServerHandler
	c.maxRequestBodyBytes = c.MaxRequestBodyBytes

	return nil
}

func DefaultBuildHandlerChain(ctx context.Context, apiHandler http.Handler, c *Server) http.Handler {
	handler := filters.TrackCompleted(apiHandler)

	if authz, ok := options.AuthzFrom(ctx); ok {
		handler = filters.WithAuthorization(handler, authz.Authorizer())
		handler = filters.TrackStarted(handler, "authorization")

		handler = filters.TrackCompleted(handler)
		handler = filters.WithImpersonation(handler, authz.Authorizer())
		handler = filters.TrackStarted(handler, "impersonation")
	}

	failedHandler := filters.Unauthorized()
	//failedHandler = filters.WithFailedAuthenticationAudit(failedHandler, c.AuditBackend, c.AuditPolicyChecker)

	failedHandler = filters.TrackCompleted(failedHandler)

	if authn, ok := options.AuthnFrom(ctx); ok {
		handler = filters.TrackCompleted(handler)
		handler = filters.WithAuthentication(handler, authn.Authenticator(), failedHandler, authn.APIAudiences())
		handler = filters.TrackStarted(handler, "authentication")
	}

	if sm, ok := options.SessionManagerFrom(ctx); ok {
		handler = session.WithSession(handler, sm)
	}

	handler = filters.WithCORS(handler, c.CorsAllowedOriginList, nil, nil, nil, "true")

	// WithTimeoutForNonLongRunningRequests will call the rest of the request handling in a go-routine with the
	// context with deadline. The go-routine can keep running, while the timeout logic will return a timeout to the client.
	handler = filters.WithTimeoutForNonLongRunningRequests(handler, c.LongRunningFunc)

	handler = filters.WithRequestDeadline(handler, c.LongRunningFunc, c.RequestTimeout)
	handler = filters.WithWaitGroup(handler, c.LongRunningFunc, c.HandlerChainWaitGroup)
	handler = filters.WithRequestInfo(handler, c.RequestInfoResolver)
	//if c.SecureServing != nil && c.GoawayChance > 0 {
	//	handler = filters.WithProbabilisticGoaway(handler, c.GoawayChance)
	//}
	//handler = filters.WithAuditAnnotations(handler, c.AuditBackend, c.AuditPolicyChecker)
	//handler = filters.WithWarningRecorder(handler)
	handler = filters.WithCacheControl(handler)
	handler = filters.WithHSTS(handler, c.HSTSDirectives)
	handler = filters.WithRequestReceivedTimestamp(handler)
	handler = filters.WithPanicRecovery(handler, c.RequestInfoResolver)
	return handler
}

func NewRequestInfoResolver(c *Server) *apirequest.RequestInfoFactory {
	return &apirequest.RequestInfoFactory{}
}

func (c *Server) HostPort() (string, int, error) {
	if c == nil || c.Listener == nil {
		return "", 0, fmt.Errorf("no listener found")
	}
	addr := c.Listener.Addr().String()
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return "", 0, fmt.Errorf("failed to get port from listener address %q: %v", addr, err)
	}
	port, err := utilsnet.ParsePort(portStr, true)
	if err != nil {
		return "", 0, fmt.Errorf("invalid non-numeric port %q", portStr)
	}
	return host, port, nil
}

// APIServerHandlers holds the different http.Handlers used by the API server.
// This includes the full handler chain, the director (which chooses between gorestful and nonGoRestful,
// the gorestful handler (used for the API) which falls through to the nonGoRestful handler on unregistered paths,
// and the nonGoRestful handler (which can contain a fallthrough of its own)
// FullHandlerChain -> Director -> {GoRestfulContainer,NonGoRestfulMux} based on inspection of registered web services
type APIServerHandler struct {
	// FullHandlerChain is the one that is eventually served with.  It should include the full filter
	// chain and then call the Director.
	FullHandlerChain http.Handler
	// The registered APIs.  InstallAPIs uses this.  Other servers probably shouldn't access this directly.
	GoRestfulContainer *restful.Container
	// NonGoRestfulMux is the final HTTP handler in the chain.
	// It comes after all filters and the API handling
	// This is where other servers can attach handler to various parts of the chain.
	NonGoRestfulMux *mux.PathRecorderMux
}

// HandlerChainBuilderFn is used to wrap the GoRestfulContainer handler using the provided handler chain.
// It is normally used to apply filtering like authentication and authorization
type HandlerChainBuilderFn func(ctx context.Context, apiHandler http.Handler) http.Handler

func NewAPIServerHandler(ctx context.Context, handlerChainBuilder HandlerChainBuilderFn) *APIServerHandler {

	gorestfulContainer := restful.NewContainer()
	//gorestfulContainer.ServeMux = http.NewServeMux()
	gorestfulContainer.Router(restful.CurlyRouter{}) // e.g. for proxy/{kind}/{name}/{*}
	gorestfulContainer.RecoverHandler(func(panicReason interface{}, httpWriter http.ResponseWriter) {
		logStackOnRecover(panicReason, httpWriter)
	})
	gorestfulContainer.ServiceErrorHandler(func(serviceErr restful.ServiceError, request *restful.Request, response *restful.Response) {
		serviceErrorHandler(serviceErr, request, response)
	})

	nonGoRestfulMux := mux.NewPathRecorderMux(proc.NameFrom(ctx), gorestfulContainer.ServeMux)

	return &APIServerHandler{
		FullHandlerChain:   handlerChainBuilder(ctx, gorestfulContainer.ServeMux),
		GoRestfulContainer: gorestfulContainer,
		NonGoRestfulMux:    nonGoRestfulMux,
	}
}

// ListedPaths returns the paths that should be shown under /
func (a *APIServerHandler) ListedPaths() []string {
	var handledPaths []string
	// Extract the paths handled using restful.WebService
	for _, ws := range a.GoRestfulContainer.RegisteredWebServices() {
		handledPaths = append(handledPaths, ws.RootPath())
	}
	handledPaths = append(handledPaths, a.NonGoRestfulMux.ListedPaths()...)
	sort.Strings(handledPaths)

	return handledPaths
}

//TODO: Unify with RecoverPanics?
func logStackOnRecover(panicReason interface{}, w http.ResponseWriter) {
	var buffer bytes.Buffer
	buffer.WriteString(fmt.Sprintf("recover from panic situation: - %v\r\n", panicReason))
	for i := 2; ; i++ {
		_, file, line, ok := rt.Caller(i)
		if !ok {
			break
		}
		buffer.WriteString(fmt.Sprintf("    %s:%d\r\n", file, line))
	}
	klog.Errorln(buffer.String())

	headers := http.Header{}
	if ct := w.Header().Get("Content-Type"); len(ct) > 0 {
		headers.Set("Accept", ct)
	}
	responsewriters.Error(apierrors.NewGenericServerResponse(
		http.StatusInternalServerError, "", "", "", 0, false),
		w, &http.Request{Header: headers})
}

func serviceErrorHandler(serviceErr restful.ServiceError, request *restful.Request, resp *restful.Response) {
	responsewriters.Error(
		apierrors.NewGenericServerResponse(serviceErr.Code, "", "", serviceErr.Message, 0, false),
		resp,
		request.Request,
	)
}

// ServeHTTP makes it an http.Handler
func (a *APIServerHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.FullHandlerChain.ServeHTTP(w, r)
}

// Serve runs the secure http server. It fails only if certificates cannot be loaded or the initial listen call fails.
// The actual server loop (stoppable by closing stopCh) runs in a go routine, i.e. Serve does not block.
// It returns a stoppedCh that is closed when all non-hijacked active requests have been processed.
func (s *Server) Serve(handler http.Handler, shutdownTimeout time.Duration, stopCh <-chan struct{}) (<-chan struct{}, error) {
	if s.Listener == nil {
		return nil, fmt.Errorf("listener must not be nil")
	}

	secureServer := &http.Server{
		Addr:           s.Listener.Addr().String(),
		Handler:        handler,
		MaxHeaderBytes: 1 << 20,
		//TLSConfig:      tlsConfig,
	}

	klog.Infof("Serving securely on %s", secureServer.Addr)
	return RunServer(secureServer, s.Listener, shutdownTimeout, stopCh)
}

// RunServer spawns a go-routine continuously serving until the stopCh is
// closed.
// It returns a stoppedCh that is closed when all non-hijacked active requests
// have been processed.
// This function does not block
// TODO: make private when insecure serving is gone from the kube-apiserver
func RunServer(
	server *http.Server,
	ln net.Listener,
	shutDownTimeout time.Duration,
	stopCh <-chan struct{},
) (<-chan struct{}, error) {
	if ln == nil {
		return nil, fmt.Errorf("listener must not be nil")
	}

	// Shutdown server gracefully.
	stoppedCh := make(chan struct{})
	go func() {
		<-stopCh
		ctx, cancel := context.WithTimeout(context.Background(), shutDownTimeout)
		server.Shutdown(ctx)
		cancel()
		close(stoppedCh)
	}()

	go func() {
		defer utilruntime.HandleCrash()

		var listener net.Listener
		listener = tcpKeepAliveListener{ln}
		if server.TLSConfig != nil {
			listener = tls.NewListener(listener, server.TLSConfig)
		}

		err := server.Serve(listener)

		msg := fmt.Sprintf("Stopped listening on %s", ln.Addr().String())
		select {
		case <-stopCh:
			klog.Info(msg)
		default:
			panic(fmt.Sprintf("%s due to error: %v", msg, err))
		}
	}()

	return stoppedCh, nil
}

// tcpKeepAliveListener sets TCP keep-alive timeouts on accepted
// connections. It's used by ListenAndServe and ListenAndServeTLS so
// dead TCP connections (e.g. closing laptop mid-download) eventually
// go away.
//
// Copied from Go 1.7.2 net/http/server.go
type tcpKeepAliveListener struct {
	net.Listener
}

func (ln tcpKeepAliveListener) Accept() (net.Conn, error) {
	c, err := ln.Listener.Accept()
	if err != nil {
		return nil, err
	}
	if tc, ok := c.(*net.TCPConn); ok {
		tc.SetKeepAlive(true)
		tc.SetKeepAlivePeriod(defaultKeepAlivePeriod)
	}
	return c, nil
}

func (s *Server) Start(stopCh <-chan struct{}, done chan struct{}) error {
	delayedStopCh := make(chan struct{})

	// close socket after delayed stopCh
	stoppedCh, err := s.Serve(s.Handler, s.ShutdownTimeout, delayedStopCh)
	if err != nil {
		return err
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
		// wait for stoppedCh that is closed when the graceful termination (server.Shutdown) is finished.
		<-stoppedCh

		// Wait for all requests to finish, which are bounded by the RequestTimeout variable.
		s.HandlerChainWaitGroup.Wait()

		close(done)
	}()
	return nil
}
