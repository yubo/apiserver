package main

import (
	"context"
	"net/http"
	"time"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"
	"github.com/yubo/apiserver/examples/tracing/api"
	"github.com/yubo/apiserver/pkg/rest"
	"github.com/yubo/golib/net/rpc"
	"github.com/yubo/golib/scheme"
	"k8s.io/klog/v2"
)

type demo struct {
	container rest.GoRestfulContainer
}

func (p demo) install() {
	rest.SwaggerTagRegister("tracing", "tracing demo")

	rest.WsRouteBuild(&rest.WsOption{
		Path:               "/tracing",
		Produces:           []string{"*/*"},
		Consumes:           []string{"*/*"},
		Tags:               []string{"tracing"},
		GoRestfulContainer: p.container,
		Routes: []rest.WsRoute{{
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
		}},
	})
}

func delay() {
	time.Sleep(time.Millisecond * 100)
}

// a -> a1
func (p demo) a(w http.ResponseWriter, req *http.Request) {
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
func (p demo) b(w http.ResponseWriter, req *http.Request) error {
	klog.Info("b entering")
	sp, ctx := opentracing.StartSpanFromContext(req.Context(), "helo.tracing.b")
	defer sp.Finish()

	sp.LogFields(log.String("msg", "from b"))
	delay()

	// call b1
	c, err := rest.RESTClientFor(&rest.Config{
		Host:          req.Host,
		ContentConfig: rest.ContentConfig{NegotiatedSerializer: scheme.Codecs},
	})
	if err != nil {
		return err
	}

	err = c.Get().Prefix("traceing", "b1").Do(ctx).Error()
	klog.Infof("b leaving err %v", err)

	return err
}

func (p demo) b1(w http.ResponseWriter, req *http.Request) {
	klog.Info("b1 entering")
	sp, _ := opentracing.StartSpanFromContext(
		req.Context(), "helo.tracing.b1",
	)
	defer sp.Finish()

	sp.LogFields(log.String("msg", "from b1"))
	delay()
}

func (p demo) c(w http.ResponseWriter, req *http.Request) (string, error) {
	ctx := req.Context()
	sp, _ := opentracing.StartSpanFromContext(ctx, "helo.tracing.c")
	defer sp.Finish()

	sp.LogFields(log.String("msg", "from c"))
	delay()

	//time.Sleep(time.Second * 1)
	conn, err := rpc.DialRr(ctx, "127.0.0.1:8081", false)
	if err != nil {
		klog.Errorf("Dial err %v\n", err)
		return "", err
	}
	defer conn.Close()

	resp, err := api.NewServiceClient(conn).C1(ctx, &api.Request{Name: "tom"})
	if err != nil {
		return "", err
	}

	return resp.Message, nil
}

type grpcserver struct {
	api.UnimplementedServiceServer
}

func (s *grpcserver) C1(ctx context.Context, in *api.Request) (*api.Response, error) {
	klog.Infof("receive req : %s \n", in)

	sp, _ := opentracing.StartSpanFromContext(ctx, "helo.tracing.C1")
	defer sp.Finish()

	sp.LogFields(log.String("msg", "from C1"))

	return &api.Response{Message: "Hello " + in.Name}, nil
}
