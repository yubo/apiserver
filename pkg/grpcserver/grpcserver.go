package grpcserver

import (
	"context"

	"github.com/yubo/apiserver/components/dbus"
	"github.com/yubo/apiserver/pkg/config/configgrpc"
	"github.com/yubo/apiserver/pkg/proc"
	v1 "github.com/yubo/apiserver/pkg/proc/api/v1"
	"github.com/yubo/golib/util"
	"github.com/yubo/golib/util/validation/field"
	"go.opentelemetry.io/otel"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"k8s.io/klog/v2"
)

const moduleName = "grpc"

func Register(opts ...proc.ModuleOption) {
	o := &proc.ModuleOptions{
		Proc: proc.DefaultProcess,
	}
	for _, v := range opts {
		v(o)
	}

	module := &grpcServer{name: moduleName}
	hookOps := []v1.HookOps{{
		Hook:        module.init,
		Owner:       moduleName,
		HookNum:     v1.ACTION_START,
		Priority:    v1.PRI_SYS_INIT,
		SubPriority: v1.PRI_M_GRPC,
	}, {
		Hook:        module.start,
		Owner:       moduleName,
		HookNum:     v1.ACTION_START,
		Priority:    v1.PRI_SYS_START,
		SubPriority: v1.PRI_M_GRPC,
	}, {
		Hook:        module.stop,
		Owner:       moduleName,
		HookNum:     v1.ACTION_STOP,
		Priority:    v1.PRI_SYS_START,
		SubPriority: v1.PRI_M_GRPC,
	}}

	o.Proc.RegisterHooks(hookOps)
	o.Proc.AddConfig(moduleName, newConfig(), proc.WithConfigGroup(moduleName))
}

func newConfig() *configgrpc.GRPCServerSettings {
	return &configgrpc.GRPCServerSettings{}
}

type grpcServer struct {
	name   string
	config *configgrpc.GRPCServerSettings
	grpc   *grpc.Server
	ctx    context.Context
	cancel context.CancelFunc
}

func (p *grpcServer) init(ctx context.Context) error {
	p.ctx, p.cancel = context.WithCancel(ctx)

	cf := newConfig()
	if err := proc.ReadConfig(p.name, cf); err != nil {
		return field.NotFound(field.NewPath(p.name), cf)
	}
	p.config = cf

	// TODO:
	tracerProvider := otel.GetTracerProvider()
	propagators := otel.GetTextMapPropagator()

	opts, err := cf.ToServerOption(tracerProvider, propagators)
	if err != nil {
		return err
	}

	// grpc api
	p.grpc = grpc.NewServer(opts...)

	dbus.RegisterGrpcServer(p.grpc)
	return nil
}

func (p *grpcServer) start(ctx context.Context) error {
	cf := p.config
	server := p.grpc

	if util.AddrIsDisable(cf.Endpoint) {
		klog.InfoS("grpcServer is disabled", "grpc.addr", cf.Endpoint)
		return nil
	}

	ln, err := cf.ToListener()
	if err != nil {
		return err
	}
	klog.InfoS("grpc Listen", "address", cf.Endpoint)

	reflection.Register(server)

	proc.WaitGroupAdd(p.ctx, func() {
		if err := server.Serve(ln); err != nil {
			klog.Fatal(err)
			return
		}
	})

	go func() {
		<-p.ctx.Done()
		server.GracefulStop()
	}()

	return nil

}

func (p *grpcServer) stop(ctx context.Context) error {
	p.cancel()
	return nil
}
