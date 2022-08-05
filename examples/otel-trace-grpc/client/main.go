package main

import (
	"context"
	"flag"
	"log"

	"examples/otel-trace-grpc/api"

	"github.com/yubo/apiserver/pkg/config/configgrpc"
	"github.com/yubo/apiserver/pkg/config/configtls"
	"github.com/yubo/apiserver/pkg/grpcclient"
	"github.com/yubo/golib/util"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	oteltrace "go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
)

const (
	otelAgentAddr = "0.0.0.0:4317"
	traceName     = "examples/otel-trace-grpc/client"
)

var (
	apiversion = "1"
)

func initProvider() func() {
	ctx := context.Background()

	traceClient := otlptracegrpc.NewClient(
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithEndpoint(otelAgentAddr),
		otlptracegrpc.WithDialOption(grpc.WithBlock()))
	traceExp, err := otlptrace.New(ctx, traceClient)
	handleErr(err, "Failed to create the collector trace exporter")

	res, err := resource.New(ctx,
		resource.WithFromEnv(),
		resource.WithProcess(),
		resource.WithTelemetrySDK(),
		resource.WithHost(),
		resource.WithAttributes(
			// the service name used to display traces in backends
			semconv.ServiceNameKey.String("otel-trace.client"),
		),
	)
	handleErr(err, "failed to create resource")

	bsp := sdktrace.NewBatchSpanProcessor(traceExp)
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(bsp),
	)

	// set global propagator to tracecontext (the default is no-op).
	otel.SetTextMapPropagator(propagation.TraceContext{})
	otel.SetTracerProvider(tracerProvider)

	return func() {
		if err := tracerProvider.Shutdown(ctx); err != nil {
			otel.Handle(err)
		}
	}
}

func makeRequest(ctx context.Context, version int) {
	conn, err := grpcclient.Dial(ctx, &configgrpc.GRPCClientSettings{
		Endpoint: "127.0.0.1:8081",
		TLSSetting: configtls.TLSClientSetting{
			Insecure: true,
		},
	})
	if err != nil {
		handleErr(err, "failed to grpc dial")
	}
	defer conn.Close()

	in := &api.UserGetInput{Name: util.String("tom")}
	var user *api.User
	switch version {
	case 1:
		user, err = api.NewServiceClient(conn).GetUserV1(ctx, in)
	case 2:
		user, err = api.NewServiceClient(conn).GetUserV2(ctx, in)
	case 3:
		user, err = api.NewServiceClient(conn).GetUserV3(ctx, in)
	}

	if err != nil {
		handleErr(err, "failed to grpc call")
	}

	log.Printf("get user %s", user)
}

func handleErr(err error, message string) {
	if err != nil {
		log.Fatalf("%s: %v", message, err)
	}
}

func main() {
	version := flag.Int("version", 3, "api version[1~3]")
	flag.Parse()

	shutdown := initProvider()
	defer shutdown()

	tracer := otel.Tracer(traceName, oteltrace.WithInstrumentationVersion("0.1"))
	ctx, span := tracer.Start(context.Background(), "ExecuteRequest")
	log.Printf("tracer.Start traceID: %s", span.SpanContext().TraceID())
	makeRequest(ctx, *version)
	span.End()
}
