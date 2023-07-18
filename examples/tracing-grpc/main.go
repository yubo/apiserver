// GetUserV1
// GetUserV2 --> func --> GetUserV1
// GetUserV3 --> grpc --> GetUserV1
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"examples/tracing-grpc/api"

	"github.com/yubo/apiserver/components/cli"
	"github.com/yubo/apiserver/components/dbus"
	"github.com/yubo/apiserver/pkg/config/configgrpc"
	"github.com/yubo/apiserver/pkg/config/configtls"
	"github.com/yubo/apiserver/pkg/grpcclient"
	"github.com/yubo/apiserver/pkg/proc"
	"github.com/yubo/apiserver/pkg/tracing"
	"github.com/yubo/golib/util"
	"go.opentelemetry.io/otel/attribute"

	_ "github.com/yubo/apiserver/pkg/grpcserver/register"
	_ "github.com/yubo/apiserver/pkg/tracing/register"
)

func main() {
	cmd := proc.NewRootCmd(proc.WithRun(start))
	code := cli.Run(cmd)
	os.Exit(code)
}

func start(ctx context.Context) error {
	if grpc, err := dbus.GetGrpcServer(); err != nil {
		return fmt.Errorf("unable to get grpc server from the context")
	} else {
		api.RegisterServiceServer(grpc, &grpcserver{})
	}
	return nil
}

type grpcserver struct {
	api.UnimplementedServiceServer
}

func (s *grpcserver) GetUserV1(ctx context.Context, in *api.UserGetInput) (*api.User, error) {
	_, span := tracing.Start(ctx, "GetUserV1", attribute.String("name", util.StringValue(in.Name)))
	defer span.End(100 * time.Millisecond)

	return &api.User{Name: in.Name}, nil
}

func (s *grpcserver) GetUserV2(ctx context.Context, in *api.UserGetInput) (*api.User, error) {
	_, span := tracing.Start(ctx, "GetUserV2", attribute.String("name", util.StringValue(in.Name)))
	defer span.End(100 * time.Millisecond)

	return s.GetUserV1(ctx, in)
}

func (s *grpcserver) GetUserV3(ctx context.Context, in *api.UserGetInput) (*api.User, error) {
	_, span := tracing.Start(ctx, "GetUserV3", attribute.String("name", util.StringValue(in.Name)))
	defer span.End(100 * time.Millisecond)

	conn, err := grpcclient.Dial(ctx, &configgrpc.GRPCClientSettings{
		Endpoint: "127.0.0.1:8081",
		TLSSetting: configtls.TLSClientSetting{
			Insecure: true,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("Dial err %v\n", err)
	}
	defer conn.Close()

	return api.NewServiceClient(conn).GetUserV1(ctx, in)
}
