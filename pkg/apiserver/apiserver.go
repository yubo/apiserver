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
	"github.com/yubo/apiserver/pkg/request"
	"github.com/yubo/apiserver/pkg/responsewriters"
	apierrors "github.com/yubo/golib/api/errors"
	"github.com/yubo/golib/proc"
	utilruntime "github.com/yubo/golib/util/runtime"
	"github.com/yubo/golib/util/sets"
	utilwaitgroup "github.com/yubo/golib/util/waitgroup"
	"k8s.io/klog/v2"
)

const (
	defaultKeepAlivePeriod = 3 * time.Minute
)

// same as http.Handle()
func (p *apiserver) Handle(pattern string, handler http.Handler) {
	p.handler.GoRestfulContainer.Handle(pattern, handler)
}
func (p *apiserver) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	p.Handle(pattern, http.HandlerFunc(handler))
}
func (p *apiserver) UnlistedHandle(pattern string, handler http.Handler) {
	p.handler.NonGoRestfulMux.UnlistedHandle(pattern, handler)
}
func (p *apiserver) UnlistedHandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	p.UnlistedHandle(pattern, http.HandlerFunc(handler))
}

func (p *apiserver) Add(service *restful.WebService) *restful.Container {
	return p.handler.GoRestfulContainer.Add(service)
}

func (p *apiserver) Filter(filter restful.FilterFunction) {
	p.handler.GoRestfulContainer.Filter(filter)
}

func (p *apiserver) serverInit() (err error) {
	c := p.config

	if c.Enabled {
		addr := net.JoinHostPort(c.Host, strconv.Itoa(c.Port))
		c.Listener, c.Port, err = createListener(c.Network, addr, net.ListenConfig{})
		if err != nil {
			return fmt.Errorf("failed to create listener: %v", err)
		}
	}

	p.stoppedCh = make(chan struct{})
	p.apiServerID = proc.NameFrom(p.ctx) + "-" + uuid.New().String()
	p.handlerChainWaitGroup = new(utilwaitgroup.SafeWaitGroup)
	p.longRunningFunc = filters.BasicLongRunningRequestCheck(sets.NewString("watch"), sets.NewString())
	p.requestInfoResolver = NewRequestInfoResolver(c)

	p.handler = NewAPIServerHandler(p.ctx, func(handler http.Handler) http.Handler {
		return DefaultBuildHandlerChain(p.ctx, handler, p)
	})

	return nil
}

func DefaultBuildHandlerChain(ctx context.Context, apiHandler http.Handler, p *apiserver) http.Handler {
	handler := apiHandler

	if authz, ok := options.AuthzFrom(ctx); ok {
		handler = filters.TrackCompleted(apiHandler)
		handler = filters.WithAuthorization(handler, authz.Authorizer())
		handler = filters.TrackStarted(handler, "authorization")

		handler = filters.TrackCompleted(handler)
		handler = filters.WithImpersonation(handler, authz.Authorizer())
		handler = filters.TrackStarted(handler, "impersonation")
	}

	handler = filters.TrackCompleted(handler)
	handler = filters.WithAudit(handler, p.AuditBackend, p.AuditPolicyChecker, p.longRunningFunc)
	handler = filters.TrackStarted(handler, "audit")

	failedHandler := filters.Unauthorized()
	failedHandler = filters.WithFailedAuthenticationAudit(failedHandler, p.AuditBackend, p.AuditPolicyChecker)

	failedHandler = filters.TrackCompleted(failedHandler)

	if authn, ok := options.AuthnFrom(ctx); ok {
		handler = filters.TrackCompleted(handler)
		handler = filters.WithAuthentication(handler, authn.Authenticator(), failedHandler)
		handler = filters.TrackStarted(handler, "authentication")
	}

	if sm, ok := options.SessionManagerFrom(ctx); ok {
		handler = filters.WithSession(handler, sm)
	}

	handler = filters.WithCORS(handler, p.config.CorsAllowedOriginList, nil, nil, nil, "true")

	// WithTimeoutForNonLongRunningRequests will call the rest of the request handling in a go-routine with the
	// context with deadline. The go-routine can keep running, while the timeout logic will return a timeout to the client.
	handler = filters.WithTimeoutForNonLongRunningRequests(handler, p.longRunningFunc)

	handler = filters.WithRequestDeadline(handler, p.AuditBackend, p.AuditPolicyChecker, p.longRunningFunc, p.config.requestTimeout)
	handler = filters.WithWaitGroup(handler, p.longRunningFunc, p.handlerChainWaitGroup)
	handler = filters.WithRequestInfo(handler, p.requestInfoResolver)
	//if c.SecureServing != nil && c.GoawayChance > 0 {
	//	handler = filters.WithProbabilisticGoaway(handler, c.GoawayChance)
	//}
	handler = filters.WithAuditAnnotations(handler, p.AuditBackend, p.AuditPolicyChecker)
	handler = filters.WithWarningRecorder(handler)
	handler = filters.WithCacheControl(handler)
	handler = filters.WithHSTS(handler, p.config.HSTSDirectives)
	handler = filters.WithRequestReceivedTimestamp(handler)
	handler = filters.WithPanicRecovery(handler, p.requestInfoResolver)
	return handler
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
type HandlerChainBuilderFn func(apiHandler http.Handler) http.Handler

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
		FullHandlerChain:   handlerChainBuilder(gorestfulContainer.ServeMux),
		GoRestfulContainer: gorestfulContainer,
		NonGoRestfulMux:    nonGoRestfulMux,
	}
}

func serviceErrorHandler(serviceErr restful.ServiceError, request *restful.Request, resp *restful.Response) {
	responsewriters.Error(
		apierrors.NewGenericServerResponse(serviceErr.Code, "", "", serviceErr.Message, 0, false),
		resp,
		request.Request,
	)
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
func (s *apiserver) Start(stopCh <-chan struct{}, done chan struct{}) error {
	if !s.config.Enabled {
		klog.Infof("apiserver is disabled")
		close(done)
		return nil
	}

	delayedStopCh := make(chan struct{})

	// close socket after delayed stopCh
	stoppedCh, err := s.Serve(s.handler, s.config.shutdownTimeout, delayedStopCh)
	if err != nil {
		return err
	}

	go func() {
		<-stopCh
		time.Sleep(s.config.shutdownDelayDuration)
		close(delayedStopCh)
	}()

	go func() {
		<-stopCh

		// wait for the delayed stopCh before closing the handler chain (it rejects everything after Wait has been called).
		<-delayedStopCh
		// wait for stoppedCh that is closed when the graceful termination (server.Shutdown) is finished.
		<-stoppedCh

		// Wait for all requests to finish, which are bounded by the RequestTimeout variable.
		s.handlerChainWaitGroup.Wait()

		close(done)
	}()
	return nil
}

// Serve runs the secure http server. It fails only if certificates cannot be loaded or the initial listen call fails.
// The actual server loop (stoppable by closing stopCh) runs in a go routine, i.e. Serve does not block.
// It returns a stoppedCh that is closed when all non-hijacked active requests have been processed.
func (s *apiserver) Serve(handler http.Handler, shutdownTimeout time.Duration, stopCh <-chan struct{}) (<-chan struct{}, error) {
	if s.config.Listener == nil {
		return nil, fmt.Errorf("listener must not be nil")
	}

	server := &http.Server{
		Addr:           s.config.Listener.Addr().String(),
		Handler:        handler,
		MaxHeaderBytes: 1 << 20,
		//TLSConfig:      tlsConfig,
		ConnContext: func(ctx context.Context, c net.Conn) context.Context {
			// Store the connection in the context so requests can reference it if needed
			return request.WithConn(ctx, c)
		},
	}

	klog.Infof("Serving on %s", server.Addr)
	return RunServer(server, s.config.Listener, shutdownTimeout, stopCh)
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

// ServeHTTP makes it an http.Handler
func (a *APIServerHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.FullHandlerChain.ServeHTTP(w, r)
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

func createListener(network, addr string, config net.ListenConfig) (net.Listener, int, error) {
	if len(network) == 0 {
		network = "tcp"
	}

	ln, err := config.Listen(context.TODO(), network, addr)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to listen on %v: %v", addr, err)
	}

	// get port
	tcpAddr, ok := ln.Addr().(*net.TCPAddr)
	if !ok {
		ln.Close()
		return nil, 0, fmt.Errorf("invalid listen address: %q", ln.Addr().String())
	}

	return ln, tcpAddr.Port, nil
}
