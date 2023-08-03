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

func Register(opts ...proc.ModuleOption) {
	o := &proc.ModuleOptions{
		Proc: proc.DefaultProcess,
	}
	for _, v := range opts {
		v(o)
	}

	module := &serverModule{name: moduleName}
	hookOps := []v1.HookOps{{
		Hook:        module.init,
		Owner:       moduleName,
		HookNum:     v1.ACTION_START,
		Priority:    v1.PRI_SYS_INIT,
		SubPriority: v1.PRI_M_HTTP,
	}, {
		Hook:        module.start,
		Owner:       moduleName,
		HookNum:     v1.ACTION_START,
		Priority:    v1.PRI_SYS_START,
		SubPriority: v1.PRI_M_HTTP,
	}, {
		Hook:        module.stop,
		Owner:       moduleName,
		HookNum:     v1.ACTION_STOP,
		Priority:    v1.PRI_SYS_START,
		SubPriority: v1.PRI_M_HTTP,
	}}

	o.Proc.RegisterHooks(hookOps)

	cf := newConfig()
	o.Proc.AddConfig("generic", cf.GenericServerRunOptions, proc.WithConfigGroup("generic"))
	o.Proc.AddConfig("secureServing", cf.SecureServing, proc.WithConfigGroup("secureServing"))
	o.Proc.AddConfig("insecureServing", cf.InsecureServing, proc.WithConfigGroup("insecureServing"))
	o.Proc.AddConfig("audit", cf.Audit, proc.WithConfigGroup("audit"))
	o.Proc.AddConfig("feature", cf.Features, proc.WithConfigGroup("feature"))
	o.Proc.AddConfig("authentication", cf.Authentication, proc.WithConfigGroup("authentication"))
	o.Proc.AddConfig("authorization", cf.Authorization, proc.WithConfigGroup("authorization"))
}

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
