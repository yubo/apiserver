package grpcserver

import (
	"context"
	"net"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/yubo/apiserver/pkg/options"
	"github.com/yubo/golib/configer"
	"github.com/yubo/golib/net/rpc"
	"github.com/yubo/golib/proc"
	"github.com/yubo/golib/util"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
	"k8s.io/klog/v2"
)

const (
	moduleName = "grpc"
)

type config struct {
	Addr           string `json:"addr"`
	MaxRecvMsgSize int    `json:"maxRecvMsgSize"`
}

type grpcServer struct {
	name   string
	config *config
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

func (p *grpcServer) init(ctx context.Context) (err error) {
	c := configer.ConfigerMustFrom(ctx)
	p.ctx, p.cancel = context.WithCancel(ctx)

	cf := &config{}
	if err := c.Read(p.name, cf); err != nil {
		return err
	}
	p.config = cf

	// grpc api
	p.grpc = newServer(cf, grpc.UnaryInterceptor(interceptor))
	// TODO: lookup authn & authz

	options.WithGrpcServer(ctx, p.grpc)
	return nil
}

func (p *grpcServer) start(ctx context.Context) error {
	cf := p.config
	server := p.grpc

	if util.AddrIsDisable(cf.Addr) {
		return nil
	}

	ln, err := net.Listen(util.CleanSockFile(util.ParseAddr(cf.Addr)))
	if err != nil {
		return err
	}
	klog.V(5).Infof("grpcServer Listen addr %s", cf.Addr)

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

func newServer(cf *config, opt ...grpc.ServerOption) *grpc.Server {
	if cf.MaxRecvMsgSize > 0 {
		klog.V(5).Infof("set grpc server max recv msg size %s",
			util.ByteSize(cf.MaxRecvMsgSize).HumanReadable())
		opt = append(opt, grpc.MaxRecvMsgSize(cf.MaxRecvMsgSize))
	}

	return grpc.NewServer(opt...)
}

func interceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {

	if opentracing.IsGlobalTracerRegistered() {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			md = metadata.New(nil)
		}
		tr := opentracing.GlobalTracer()
		spanContext, _ := tr.Extract(opentracing.TextMap, rpc.TextMapCarrier{MD: md})
		sp := tr.StartSpan(info.FullMethod,
			ext.RPCServerOption(spanContext), ext.SpanKindRPCServer)
		defer sp.Finish()
		ctx = opentracing.ContextWithSpan(ctx, sp)
	}

	return handler(ctx, req)
}

func Register() {
	proc.RegisterHooks(hookOps)
}
