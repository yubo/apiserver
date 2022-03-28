package grpcserver

import (
	"context"

	"github.com/yubo/apiserver/pkg/config/configgrpc"
	"github.com/yubo/apiserver/pkg/options"
	"github.com/yubo/golib/configer"
	"github.com/yubo/golib/proc"
	"github.com/yubo/golib/util"
	"go.opentelemetry.io/otel"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"k8s.io/klog/v2"
)

const (
	moduleName = "grpc"
)

//type config struct {
//	Addr           string `json:"addr" default:":8081" description:"grpc server address"`
//	MaxRecvMsgSize int    `json:"maxRecvMsgSize" description:"the max message size in bytes the server can receive.If this is not set, gRPC uses the default 4MB."`
//}

type grpcServer struct {
	name   string
	config *configgrpc.GRPCServerSettings
	grpc   *grpc.Server
	ctx    context.Context
	cancel context.CancelFunc
}

var (
	_module = &grpcServer{name: moduleName}
	hookOps = []proc.HookOps{{
		Hook:        _module.init,
		Owner:       moduleName,
		HookNum:     proc.ACTION_START,
		Priority:    proc.PRI_SYS_INIT,
		SubPriority: options.PRI_M_GRPC,
	}, {
		Hook:        _module.start,
		Owner:       moduleName,
		HookNum:     proc.ACTION_START,
		Priority:    proc.PRI_SYS_START,
		SubPriority: options.PRI_M_GRPC,
	}, {
		Hook:        _module.stop,
		Owner:       moduleName,
		HookNum:     proc.ACTION_STOP,
		Priority:    proc.PRI_SYS_START,
		SubPriority: options.PRI_M_GRPC,
	}}
)

func (p *grpcServer) init(ctx context.Context) error {
	c := configer.ConfigerMustFrom(ctx)
	p.ctx, p.cancel = context.WithCancel(ctx)

	cf := &configgrpc.GRPCServerSettings{}
	if err := c.Read(p.name, cf); err != nil {
		return err
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

	options.WithGrpcServer(ctx, p.grpc)
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

	go func() {
		wg, _ := proc.WgFrom(p.ctx)
		wg.Add(1)
		defer wg.Add(-1)

		if err := server.Serve(ln); err != nil {
			return
		}
	}()

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

//func newServer(cf *config, opt ...grpc.ServerOption) *grpc.Server {
//	if cf.MaxRecvMsgSize > 0 {
//		klog.V(5).Infof("set grpc server max recv msg size %s",
//			util.ByteSize(cf.MaxRecvMsgSize).HumanReadable())
//		opt = append(opt, grpc.MaxRecvMsgSize(cf.MaxRecvMsgSize))
//	}
//
//	return grpc.NewServer(opt...)
//}

func Register() {
	proc.RegisterHooks(hookOps)
}
