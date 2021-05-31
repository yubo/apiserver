// this is a sample echo rest api module
package tracing

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/emicklei/go-restful"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"
	"github.com/yubo/apiserver/pkg/openapi"
	"github.com/yubo/apiserver/pkg/options"
	"github.com/yubo/golib/net/rpc"
	"k8s.io/klog/v2"
)

const (
	moduleName = "tracing"
)

type Module struct {
	Name string
	http options.HttpServer
	ctx  context.Context
}

func New(ctx context.Context) *Module {
	return &Module{
		ctx: ctx,
	}
}

func (p *Module) Start() error {
	var ok bool
	if p.http, ok = options.GenericServerFrom(p.ctx); !ok {
		return fmt.Errorf("unable to get http server from the context")
	}

	if grpc, ok := options.GrpcServerFrom(p.ctx); !ok {
		return fmt.Errorf("unable to get grpc server from the context")
	} else {
		RegisterServiceServer(grpc, &grpcserver{})
	}

	if !opentracing.IsGlobalTracerRegistered() {
		return fmt.Errorf(" opentracing is not registered")
	}

	p.installWs()
	return nil
}

func (p *Module) installWs() {
	openapi.SwaggerTagRegister("tracing", "tracing demo")

	ws := new(restful.WebService)

	openapi.WsRouteBuild(&openapi.WsOption{
		Ws:   ws.Path("/tracing").Produces("*/*").Consumes("*/*"),
		Tags: []string{"tracing"},
	}, []openapi.WsRoute{{
		Method: "GET", SubPath: "/a",
		Desc:   "a -> a1",
		Handle: p.a,
	}, {
		Method: "GET", SubPath: "/b",
		Desc:   "b -> b1",
		Handle: p.b,
	}, {
		Method: "GET", SubPath: "/b1",
		Desc:   "b1",
		Handle: p.b1,
	}, {
		Method: "GET", SubPath: "/c",
		Desc:   "c -> C1(grpc)",
		Handle: p.c,
	}})

	p.http.Add(ws)
}

func delay() {
	time.Sleep(time.Millisecond * 100)
}

// a -> a1
func (p *Module) a(w http.ResponseWriter, req *http.Request) {
	sp, ctx := opentracing.StartSpanFromContext(req.Context(), "helo.tracing.a")
	defer sp.Finish()

	sp.LogFields(log.String("msg", "from a"))
	//delay()

	a1(ctx)
}

func a1(ctx context.Context) {
	sp, _ := opentracing.StartSpanFromContext(
		ctx, "helo.tracing.a1",
	)
	defer sp.Finish()

	sp.LogFields(log.String("msg", "from a1"))
	//delay()
}

// b -> b1
func (p *Module) b(w http.ResponseWriter, req *http.Request) error {
	klog.Info("b entering")
	sp, ctx := opentracing.StartSpanFromContext(req.Context(), "helo.tracing.b")
	defer sp.Finish()

	sp.LogFields(log.String("msg", "from b"))
	delay()

	// call b1
	_, _, err := openapi.HttpRequest(&openapi.RequestOptions{
		Url:    fmt.Sprintf("http://%s/tracing/b1", req.Host),
		Method: "GET",
		Ctx:    ctx,
	})
	klog.Infof("b leaving err %v", err)

	return err
}

func (p *Module) b1(w http.ResponseWriter, req *http.Request) {
	klog.Info("b1 entering")
	sp, _ := opentracing.StartSpanFromContext(
		req.Context(), "helo.tracing.b1",
	)
	defer sp.Finish()

	sp.LogFields(log.String("msg", "from b1"))
	delay()
}

func (p *Module) c(w http.ResponseWriter, req *http.Request) (string, error) {
	ctx := req.Context()
	sp, _ := opentracing.StartSpanFromContext(ctx, "helo.tracing.c")
	defer sp.Finish()

	sp.LogFields(log.String("msg", "from c"))
	delay()

	//time.Sleep(time.Second * 1)
	conn, _, err := rpc.DialRr(ctx, "127.0.0.1:8081", false)
	if err != nil {
		klog.Errorf("Dial err %v\n", err)
		return "", err
	}
	defer conn.Close()

	resp, err := NewServiceClient(conn).C1(ctx, &Request{Name: "tom"})
	if err != nil {
		return "", err
	}

	return resp.Message, nil
}

type grpcserver struct{}

func (s *grpcserver) C1(ctx context.Context, in *Request) (*Response, error) {
	klog.Infof("receive req : %s \n", in)

	sp, _ := opentracing.StartSpanFromContext(ctx, "helo.tracing.C1")
	defer sp.Finish()

	sp.LogFields(log.String("msg", "from C1"))

	return &Response{Message: "Hello " + in.Name}, nil
}
