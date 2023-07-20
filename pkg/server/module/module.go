package server

import (
	"context"

	"github.com/yubo/apiserver/components/dbus"
	"github.com/yubo/apiserver/pkg/filters"
	"github.com/yubo/apiserver/pkg/proc"
	v1 "github.com/yubo/apiserver/pkg/proc/api/v1"
	genericapiserver "github.com/yubo/apiserver/pkg/server"
	"github.com/yubo/apiserver/pkg/util/notfoundhandler"
	utilerrors "github.com/yubo/golib/util/errors"
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

	stopCh, runCompletedCh chan struct{}
}

// init: no dep
func (p *serverModule) init(ctx context.Context) error {
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

	notFoundHandler := notfoundhandler.New(genericConfig.Serializer, filters.NoMuxAndDiscoveryIncompleteKey)

	p.server, err = genericConfig.Complete().New("apiserver", genericapiserver.NewEmptyDelegateWithCustomHandler(notFoundHandler))
	if err != nil {
		return err
	}

	dbus.RegisterAPIServer(p.server)

	return nil
}

func (p *serverModule) start(ctx context.Context) error {

	p.stopCh, p.runCompletedCh = make(chan struct{}), make(chan struct{})

	go func() {
		defer close(p.runCompletedCh)
		if err := p.server.PrepareRun().Run(p.stopCh); err != nil {
			klog.Error(err)
		}
	}()
	//if err := p.Start(); err != nil {
	//	return err
	//}

	return nil
}

func (p *serverModule) stop(ctx context.Context) error {
	close(p.stopCh)

	<-p.runCompletedCh
	return nil
}

//func (p *serverModule) Start(stopCh <-chan struct{}, done chan struct{}) error {
//	s := p.server
//
//	delayedStopCh := make(chan struct{})
//
//	// close socket after delayed stopCh
//
//	if s.SecureServingInfo != nil {
//		_, stoppedCh, err := s.SecureServingInfo.Serve(s.Handler, s.ShutdownTimeout, delayedStopCh)
//		if err != nil {
//			return err
//		}
//		s.NonLongRunningRequestWaitGroup.Add(1)
//		go func() {
//			<-stoppedCh
//			s.NonLongRunningRequestWaitGroup.Done()
//		}()
//	}
//
//	if s.InsecureServingInfo != nil {
//		_, stoppedCh, err := s.InsecureServingInfo.Serve(s.Handler, s.ShutdownTimeout, delayedStopCh)
//		if err != nil {
//			return err
//		}
//		s.NonLongRunningRequestWaitGroup.Add(1)
//		go func() {
//			<-stoppedCh
//			s.NonLongRunningRequestWaitGroup.Done()
//		}()
//	}
//
//	// Start the audit backend before any request comes in. This means we must call Backend.Run
//	// before http server start serving. Otherwise the Backend.ProcessEvents call might block.
//	// AuditBackend.Run will stop as soon as all in-flight requests are drained.
//	if s.AuditBackend != nil {
//		if err := s.AuditBackend.Run(stopCh); err != nil {
//			return fmt.Errorf("failed to run the audit backend: %v", err)
//		}
//	}
//
//	go func() {
//		<-stopCh
//		time.Sleep(s.ShutdownDelayDuration)
//		close(delayedStopCh)
//	}()
//
//	go func() {
//		<-stopCh
//
//		// wait for the delayed stopCh before closing the handler chain (it rejects everything after Wait has been called).
//		<-delayedStopCh
//
//		// Wait for all requests to finish, which are bounded by the RequestTimeout variable.
//		s.NonLongRunningRequestWaitGroup.Wait()
//
//		close(done)
//	}()
//
//	return nil
//}

func Register() {
	proc.RegisterHooks(hookOps)

	cf := newConfig()
	proc.AddConfig("generic", cf.GenericServerRunOptions, proc.WithConfigGroup("generic"))
	proc.AddConfig("secureServing", cf.SecureServing, proc.WithConfigGroup("secureServing"))
	proc.AddConfig("insecureServing", cf.InsecureServing, proc.WithConfigGroup("insecureServing"))
	proc.AddConfig("audit", cf.Audit, proc.WithConfigGroup("audit"))
	proc.AddConfig("feature", cf.Features, proc.WithConfigGroup("feature"))
	proc.AddConfig("authentication", cf.Authentication, proc.WithConfigGroup("authentication"))
	proc.AddConfig("authorization", cf.Authorization, proc.WithConfigGroup("authorization"))
}
