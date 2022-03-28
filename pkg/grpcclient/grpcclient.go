package grpcclient

import (
	"context"

	"github.com/yubo/apiserver/pkg/config/configgrpc"
	"go.opentelemetry.io/otel"
	"google.golang.org/grpc"
	"k8s.io/klog/v2"
)

func Dial(ctx context.Context, in *configgrpc.GRPCClientSettings) (*grpc.ClientConn, error) {
	cf, err := prepareConfigWithBalancer(in)
	if err != nil {
		return nil, err
	}

	tracerProvider := otel.GetTracerProvider()
	propagators := otel.GetTextMapPropagator()

	opts, err := cf.ToDialOptions(tracerProvider, propagators)
	if err != nil {
		return nil, err
	}

	klog.V(3).InfoS("grpc.Dial", "endpoint", cf.Endpoint)

	return grpc.Dial(cf.Endpoint, opts...)
}
