package server

import (
	"context"
	"time"

	"github.com/yubo/apiserver/components/dbus"
	"github.com/yubo/apiserver/pkg/filters"
	"github.com/yubo/apiserver/pkg/proc"
	v1 "github.com/yubo/apiserver/pkg/proc/api/v1"
	genericapiserver "github.com/yubo/apiserver/pkg/server"
	genericoptions "github.com/yubo/apiserver/pkg/server/options"
	"github.com/yubo/apiserver/pkg/util/notfoundhandler"
	utilerrors "github.com/yubo/golib/util/errors"
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

type serverModule struct {
	name   string
	server *genericapiserver.GenericAPIServer

	ctx       context.Context
	cancel    context.CancelFunc
	stoppedCh chan struct{}
}

// init: no dep
func (p *serverModule) init(ctx context.Context) error {
	p.ctx, p.cancel = context.WithCancel(ctx)

	cf := newConfig()
	if err := proc.ReadConfig("", cf); err != nil {
		return err
	}

	// set default options
	c, err := complete(ctx, cf)
	if err != nil {
		return err
	}

	// validate options
	if errs := c.Validate(); len(errs) > 0 {
		return utilerrors.NewAggregate(errs)
	}

	genericConfig, err := buildGenericConfig(ctx, c.Config)
	if err != nil {
		return err
	}

	p.stoppedCh = make(chan struct{})

	notFoundHandler := notfoundhandler.New(genericConfig.Serializer, filters.NoMuxAndDiscoveryIncompleteKey)

	p.server, err = genericConfig.Complete().New("apiserver", genericapiserver.NewEmptyDelegateWithCustomHandler(notFoundHandler))
	if err != nil {
		return err
	}

	dbus.RegisterAPIServer(p.server)

	return nil
}

func (p *serverModule) start(ctx context.Context) error {
	if err := p.Start(p.ctx.Done(), p.stoppedCh); err != nil {
		return err
	}

	//p.Info()

	return nil
}

func (p *serverModule) stop(ctx context.Context) error {
	p.cancel()

	<-p.stoppedCh
	return nil
}

//	func (p *serverModule) Info() {
//		if !klog.V(10).Enabled() {
//			return
//		}
//		for _, path := range p.serverConfig.Handler.ListedPaths() {
//			klog.V(1).Infof("apiserver path %s", path)
//		}
//	}
//
//	func (p *serverModule) Config() *server.Config {
//		return p.serverConfig
//	}
//
// // Add a WebService to the Container. It will detect duplicate root paths and exit in that case.
//
//	func (p *serverModule) Add(service *restful.WebService) *restful.Container {
//		return p.serverConfig.Handler.GoRestfulContainer.Add(service)
//	}
//
// // Remove a WebService from the Container.
//
//	func (p *serverModule) Remove(service *restful.WebService) error {
//		return p.serverConfig.Handler.GoRestfulContainer.Remove(service)
//	}
//
// // Handle registers the handler for the given pattern.
// // If a handler already exists for pattern, Handle panics.
//
//	func (p *serverModule) Handle(path string, handler http.Handler) {
//		p.serverConfig.Handler.NonGoRestfulMux.Handle(path, handler)
//	}
//
// // UnlistedHandle registers the handler for the given pattern, but doesn't list it.
// // If a handler already exists for pattern, Handle panics.
//
//	func (p *serverModule) UnlistedHandle(path string, handler http.Handler) {
//		p.serverConfig.Handler.NonGoRestfulMux.UnlistedHandle(path, handler)
//	}
//
// // HandlePrefix is like Handle, but matches for anything under the path.  Like a standard golang trailing slash.
//
//	func (p *serverModule) HandlePrefix(path string, handler http.Handler) {
//		p.serverConfig.Handler.NonGoRestfulMux.HandlePrefix(path, handler)
//	}
//
// // UnlistedHandlePrefix is like UnlistedHandle, but matches for anything under the path.  Like a standard golang trailing slash.
//
//	func (p *serverModule) UnlistedHandlePrefix(path string, handler http.Handler) {
//		p.serverConfig.Handler.NonGoRestfulMux.UnlistedHandlePrefix(path, handler)
//	}
//
// // ListedPaths is an alphabetically sorted list of paths to be reported at /.
//
//	func (p *serverModule) ListedPaths() []string {
//		return p.serverConfig.ListedPathProvider.ListedPaths()
//	}
//
//	func (p *serverModule) Serializer() runtime.NegotiatedSerializer {
//		return p.serverConfig.Serializer
//	}
//
// // Filter appends a container FilterFunction. These are called before dispatching
// // a http.Request to a WebService from the container
//
//	func (p *serverModule) Filter(filter restful.FilterFunction) {
//		p.serverConfig.Handler.GoRestfulContainer.Filter(filter)
//	}
func (p *serverModule) Start(stopCh <-chan struct{}, done chan struct{}) error {
	s := p.server

	delayedStopCh := make(chan struct{})

	// close socket after delayed stopCh

	if s.SecureServingInfo != nil {
		_, stoppedCh, err := s.SecureServingInfo.Serve(s.Handler, s.ShutdownTimeout, delayedStopCh)
		if err != nil {
			return err
		}
		s.NonLongRunningRequestWaitGroup.Add(1)
		go func() {
			<-stoppedCh
			s.NonLongRunningRequestWaitGroup.Done()
		}()
	}

	if s.InsecureServingInfo != nil {
		_, stoppedCh, err := s.InsecureServingInfo.Serve(s.Handler, s.ShutdownTimeout, delayedStopCh)
		if err != nil {
			return err
		}
		s.NonLongRunningRequestWaitGroup.Add(1)
		go func() {
			<-stoppedCh
			s.NonLongRunningRequestWaitGroup.Done()
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
		s.NonLongRunningRequestWaitGroup.Wait()

		close(done)
	}()

	return nil
}

func Register() {
	proc.RegisterHooks(hookOps)
	proc.AddConfig("generic", genericoptions.NewServerRunOptions(), proc.WithConfigGroup("generic"))
	proc.AddConfig("secureServing", genericoptions.NewServerRunOptions(), proc.WithConfigGroup("secureServing"))
	proc.AddConfig("insecureServing", genericoptions.NewServerRunOptions(), proc.WithConfigGroup("insecureServing"))
	proc.AddConfig("audit", genericoptions.NewServerRunOptions(), proc.WithConfigGroup("audit"))
	proc.AddConfig("feature", genericoptions.NewServerRunOptions(), proc.WithConfigGroup("feature"))
	proc.AddConfig("authentication", genericoptions.NewServerRunOptions(), proc.WithConfigGroup("authentication"))
	proc.AddConfig("authorization", genericoptions.NewServerRunOptions(), proc.WithConfigGroup("authorization"))
}
