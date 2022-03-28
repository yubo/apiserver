// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package configgrpc

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/mostynb/go-grpc-compression/snappy"
	"github.com/mostynb/go-grpc-compression/zstd"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/balancer/roundrobin"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"

	"github.com/yubo/apiserver/pkg/config/configauth"
	"github.com/yubo/apiserver/pkg/config/configcompression"
	"github.com/yubo/apiserver/pkg/config/confignet"
	"github.com/yubo/apiserver/pkg/config/configtls"
	"go.opentelemetry.io/collector/client"
)

var errMetadataNotFound = errors.New("no request metadata found")

// Allowed balancer names to be set in grpclb_policy to discover the servers.
var allowedBalancerNames = []string{roundrobin.Name, grpc.PickFirstBalancerName}

// KeepaliveClientConfig exposes the keepalive.ClientParameters to be used by the exporter.
// Refer to the original data-structure for the meaning of each parameter:
// https://godoc.org/google.golang.org/grpc/keepalive#ClientParameters
type KeepaliveClientConfig struct {
	Time                time.Duration `json:"time,omitempty"`
	Timeout             time.Duration `json:"timeout,omitempty"`
	PermitWithoutStream bool          `json:"permitWithoutStream,omitempty"`
}

// GRPCClientSettings defines common settings for a gRPC client configuration.
type GRPCClientSettings struct {
	// The target to which the exporter is going to send traces or metrics,
	// using the gRPC protocol. The valid syntax is described at
	// https://github.com/grpc/grpc/blob/master/doc/naming.md.
	Endpoint string `json:"endpoint"`

	// The compression key for supported compression types within collector.
	Compression configcompression.CompressionType `json:"compression"`

	// TLSSetting struct exposes TLS client configuration.
	TLSSetting configtls.TLSClientSetting `json:"tls,omitempty"`

	// The keepalive parameters for gRPC client. See grpc.WithKeepaliveParams.
	// (https://godoc.org/google.golang.org/grpc#WithKeepaliveParams).
	Keepalive *KeepaliveClientConfig `json:"keepalive"`

	// ReadBufferSize for gRPC client. See grpc.WithReadBufferSize.
	// (https://godoc.org/google.golang.org/grpc#WithReadBufferSize).
	ReadBufferSize int `json:"readBufferSize"`

	// WriteBufferSize for gRPC gRPC. See grpc.WithWriteBufferSize.
	// (https://godoc.org/google.golang.org/grpc#WithWriteBufferSize).
	WriteBufferSize int `json:"writeBufferSize"`

	// WaitForReady parameter configures client to wait for ready state before sending data.
	// (https://github.com/grpc/grpc/blob/master/doc/wait-for-ready.md)
	WaitForReady bool `json:"waitForReady"`

	// The headers associated with gRPC requests.
	Headers map[string]string `json:"headers"`

	// Sets the balancer in grpclbPolicy to discover the servers. Default is pickFirst.
	// https://github.com/grpc/grpc-go/blob/master/examples/features/loadBalancing/README.md
	BalancerName string `json:"balancerName"`

	// Auth configuration for outgoing RPCs.
	Auth *configauth.Authentication `json:"auth,omitempty"`
}

// not deep copy
func (p *GRPCClientSettings) Copy() *GRPCClientSettings {
	out := &GRPCClientSettings{}
	*out = *p
	return out
}

// KeepaliveServerConfig is the configuration for keepalive.
type KeepaliveServerConfig struct {
	ServerParameters  *KeepaliveServerParameters  `json:"serverParameters,omitempty"`
	EnforcementPolicy *KeepaliveEnforcementPolicy `json:"enforcementPolicy,omitempty"`
}

// KeepaliveServerParameters allow configuration of the keepalive.ServerParameters.
// The same default values as keepalive.ServerParameters are applicable and get applied by the server.
// See https://godoc.org/google.golang.org/grpc/keepalive#ServerParameters for details.
type KeepaliveServerParameters struct {
	MaxConnectionIdle     time.Duration `json:"maxConnectionIdle,omitempty"`
	MaxConnectionAge      time.Duration `json:"maxConnectionAge,omitempty"`
	MaxConnectionAgeGrace time.Duration `json:"maxConnectionAgeGrace,omitempty"`
	Time                  time.Duration `json:"time,omitempty"`
	Timeout               time.Duration `json:"timeout,omitempty"`
}

// KeepaliveEnforcementPolicy allow configuration of the keepalive.EnforcementPolicy.
// The same default values as keepalive.EnforcementPolicy are applicable and get applied by the server.
// See https://godoc.org/google.golang.org/grpc/keepalive#EnforcementPolicy for details.
type KeepaliveEnforcementPolicy struct {
	MinTime             time.Duration `json:"minTime,omitempty"`
	PermitWithoutStream bool          `json:"permitWithoutStream,omitempty"`
}

// GRPCServerSettings defines common settings for a gRPC server configuration.
type GRPCServerSettings struct {
	// Server net.Addr config. For transport only "tcp" and "unix" are valid options.
	confignet.NetAddr

	// Configures the protocol to use TLS.
	// The default value is nil, which will cause the protocol to not use TLS.
	TLSSetting *configtls.TLSServerSetting `json:"tls,omitempty"`

	// MaxRecvMsgSizeMiB sets the maximum size (in MiB) of messages accepted by the server.
	MaxRecvMsgSizeMiB uint64 `json:"maxRecvMsgSizeMib"`

	// MaxConcurrentStreams sets the limit on the number of concurrent streams to each ServerTransport.
	// It has effect only for streaming RPCs.
	MaxConcurrentStreams uint32 `json:"maxConcurrentStreams"`

	// ReadBufferSize for gRPC server. See grpc.ReadBufferSize.
	// (https://godoc.org/google.golang.org/grpc#ReadBufferSize).
	ReadBufferSize int `json:"readBufferSize"`

	// WriteBufferSize for gRPC server. See grpc.WriteBufferSize.
	// (https://godoc.org/google.golang.org/grpc#WriteBufferSize).
	WriteBufferSize int `json:"writeBufferSize"`

	// Keepalive anchor for all the settings related to keepalive.
	Keepalive *KeepaliveServerConfig `json:"keepalive,omitempty"`

	// Auth for this receiver
	Auth *configauth.Authentication `json:"auth,omitempty"`

	// Include propagates the incoming connection's metadata to downstream consumers.
	// Experimental: *NOTE* this option is subject to change or removal in the future.
	IncludeMetadata bool `json:"includeMetadata,omitempty"`
}

// SanitizedEndpoint strips the prefix of either http:// or https:// from configgrpc.GRPCClientSettings.Endpoint.
func (gcs *GRPCClientSettings) SanitizedEndpoint() string {
	switch {
	case gcs.isSchemeHTTP():
		return strings.TrimPrefix(gcs.Endpoint, "http://")
	case gcs.isSchemeHTTPS():
		return strings.TrimPrefix(gcs.Endpoint, "https://")
	default:
		return gcs.Endpoint
	}
}

func (gcs *GRPCClientSettings) isSchemeHTTP() bool {
	return strings.HasPrefix(gcs.Endpoint, "http://")
}

func (gcs *GRPCClientSettings) isSchemeHTTPS() bool {
	return strings.HasPrefix(gcs.Endpoint, "https://")
}

// ToDialOptions maps configgrpc.GRPCClientSettings to a slice of dial options for gRPC.
func (gcs *GRPCClientSettings) ToDialOptions(tracerProvider trace.TracerProvider, propagator propagation.TextMapPropagator) ([]grpc.DialOption, error) {
	var opts []grpc.DialOption
	if configcompression.IsCompressed(gcs.Compression) {
		cp, err := getGRPCCompressionName(gcs.Compression)
		if err != nil {
			return nil, err
		}
		opts = append(opts, grpc.WithDefaultCallOptions(grpc.UseCompressor(cp)))
	}

	tlsCfg, err := gcs.TLSSetting.LoadTLSConfig()
	if err != nil {
		return nil, err
	}
	cred := insecure.NewCredentials()
	if tlsCfg != nil {
		cred = credentials.NewTLS(tlsCfg)
	} else if gcs.isSchemeHTTPS() {
		cred = credentials.NewTLS(&tls.Config{})
	}
	opts = append(opts, grpc.WithTransportCredentials(cred))

	if gcs.ReadBufferSize > 0 {
		opts = append(opts, grpc.WithReadBufferSize(gcs.ReadBufferSize))
	}

	if gcs.WriteBufferSize > 0 {
		opts = append(opts, grpc.WithWriteBufferSize(gcs.WriteBufferSize))
	}

	if gcs.Keepalive != nil {
		keepAliveOption := grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                gcs.Keepalive.Time,
			Timeout:             gcs.Keepalive.Timeout,
			PermitWithoutStream: gcs.Keepalive.PermitWithoutStream,
		})
		opts = append(opts, keepAliveOption)
	}

	// TODO
	//if gcs.Auth != nil {
	//	if host.GetExtensions() == nil {
	//		return nil, fmt.Errorf("no extensions configuration available")
	//	}

	//	grpcAuthenticator, cerr := gcs.Auth.GetClientAuthenticator(host.GetExtensions())
	//	if cerr != nil {
	//		return nil, cerr
	//	}

	//	perRPCCredentials, perr := grpcAuthenticator.PerRPCCredentials()
	//	if perr != nil {
	//		return nil, err
	//	}
	//	opts = append(opts, grpc.WithPerRPCCredentials(perRPCCredentials))
	//}

	if gcs.BalancerName != "" {
		valid := ValidateBalancerName(gcs.BalancerName)
		if !valid {
			return nil, fmt.Errorf("invalid balancerName: %s", gcs.BalancerName)
		}
		opts = append(opts, grpc.WithDefaultServiceConfig(fmt.Sprintf(`{"loadBalancingPolicy":"%s"}`, gcs.BalancerName)))
	}

	// Enable OpenTelemetry observability plugin.
	opts = append(opts, grpc.WithUnaryInterceptor(otelgrpc.UnaryClientInterceptor(otelgrpc.WithTracerProvider(tracerProvider))))
	opts = append(opts, grpc.WithStreamInterceptor(otelgrpc.StreamClientInterceptor(otelgrpc.WithTracerProvider(tracerProvider))))

	return opts, nil
}

func ValidateBalancerName(balancerName string) bool {
	for _, item := range allowedBalancerNames {
		if item == balancerName {
			return true
		}
	}
	return false
}

// ToListener returns the net.Listener constructed from the settings.
func (gss *GRPCServerSettings) ToListener() (net.Listener, error) {
	return gss.Listen()
}

// ToServerOption maps configgrpc.GRPCServerSettings to a slice of server options for gRPC.
func (gss *GRPCServerSettings) ToServerOption(tracerProvider trace.TracerProvider, propagator propagation.TextMapPropagator) ([]grpc.ServerOption, error) {
	var opts []grpc.ServerOption

	if gss.TLSSetting != nil {
		tlsCfg, err := gss.TLSSetting.LoadTLSConfig()
		if err != nil {
			return nil, err
		}
		opts = append(opts, grpc.Creds(credentials.NewTLS(tlsCfg)))
	}

	if gss.MaxRecvMsgSizeMiB > 0 {
		opts = append(opts, grpc.MaxRecvMsgSize(int(gss.MaxRecvMsgSizeMiB*1024*1024)))
	}

	if gss.MaxConcurrentStreams > 0 {
		opts = append(opts, grpc.MaxConcurrentStreams(gss.MaxConcurrentStreams))
	}

	if gss.ReadBufferSize > 0 {
		opts = append(opts, grpc.ReadBufferSize(gss.ReadBufferSize))
	}

	if gss.WriteBufferSize > 0 {
		opts = append(opts, grpc.WriteBufferSize(gss.WriteBufferSize))
	}

	// The default values referenced in the GRPC docs are set within the server, so this code doesn't need
	// to apply them over zero/nil values before passing these as grpc.ServerOptions.
	// The following shows the server code for applying default grpc.ServerOptions.
	// https://github.com/grpc/grpc-go/blob/120728e1f775e40a2a764341939b78d666b08260/internal/transport/http2_server.go#L184-L200
	if gss.Keepalive != nil {
		if gss.Keepalive.ServerParameters != nil {
			svrParams := gss.Keepalive.ServerParameters
			opts = append(opts, grpc.KeepaliveParams(keepalive.ServerParameters{
				MaxConnectionIdle:     svrParams.MaxConnectionIdle,
				MaxConnectionAge:      svrParams.MaxConnectionAge,
				MaxConnectionAgeGrace: svrParams.MaxConnectionAgeGrace,
				Time:                  svrParams.Time,
				Timeout:               svrParams.Timeout,
			}))
		}
		// The default values referenced in the GRPC are set within the server, so this code doesn't need
		// to apply them over zero/nil values before passing these as grpc.ServerOptions.
		// The following shows the server code for applying default grpc.ServerOptions.
		// https://github.com/grpc/grpc-go/blob/120728e1f775e40a2a764341939b78d666b08260/internal/transport/http2_server.go#L202-L205
		if gss.Keepalive.EnforcementPolicy != nil {
			enfPol := gss.Keepalive.EnforcementPolicy
			opts = append(opts, grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
				MinTime:             enfPol.MinTime,
				PermitWithoutStream: enfPol.PermitWithoutStream,
			}))
		}
	}

	var uInterceptors []grpc.UnaryServerInterceptor
	var sInterceptors []grpc.StreamServerInterceptor

	// TODO
	//if gss.Auth != nil {
	//	authenticator, err := gss.Auth.GetServerAuthenticator(host.GetExtensions())
	//	if err != nil {
	//		return nil, err
	//	}

	//	uInterceptors = append(uInterceptors, func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	//		return authUnaryServerInterceptor(ctx, req, info, handler, authenticator.Authenticate)
	//	})
	//	sInterceptors = append(sInterceptors, func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	//		return authStreamServerInterceptor(srv, ss, info, handler, authenticator.Authenticate)
	//	})
	//}

	// Enable OpenTelemetry observability plugin.
	// TODO: Pass construct settings to have access to Tracer.
	uInterceptors = append(uInterceptors, otelgrpc.UnaryServerInterceptor(
		otelgrpc.WithTracerProvider(tracerProvider),
		// TODO: https://github.com/open-telemetry/opentelemetry-collector/issues/4030
		// otelgrpc.WithMeterProvider(settings.MeterProvider),
		otelgrpc.WithPropagators(propagator),
	))
	sInterceptors = append(sInterceptors, otelgrpc.StreamServerInterceptor(
		otelgrpc.WithTracerProvider(tracerProvider),
		// TODO: https://github.com/open-telemetry/opentelemetry-collector/issues/4030
		// otelgrpc.WithMeterProvider(settings.MeterProvider),
		otelgrpc.WithPropagators(propagator),
	))

	uInterceptors = append(uInterceptors, enhanceWithClientInformation(gss.IncludeMetadata))
	sInterceptors = append(sInterceptors, enhanceStreamWithClientInformation(gss.IncludeMetadata))

	opts = append(opts, grpc.ChainUnaryInterceptor(uInterceptors...), grpc.ChainStreamInterceptor(sInterceptors...))

	return opts, nil
}

// getGRPCCompressionName returns compression name registered in grpc.
func getGRPCCompressionName(compressionType configcompression.CompressionType) (string, error) {
	switch compressionType {
	case configcompression.Gzip:
		return gzip.Name, nil
	case configcompression.Snappy:
		return snappy.Name, nil
	case configcompression.Zstd:
		return zstd.Name, nil
	default:
		return "", fmt.Errorf("unsupported compression type %q", compressionType)
	}
}

// enhanceWithClientInformation intercepts the incoming RPC, replacing the incoming context with one that includes
// a client.Info, potentially with the peer's address.
func enhanceWithClientInformation(includeMetadata bool) func(ctx context.Context, req interface{}, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	return func(ctx context.Context, req interface{}, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		return handler(contextWithClient(ctx, includeMetadata), req)
	}
}

func enhanceStreamWithClientInformation(includeMetadata bool) func(srv interface{}, ss grpc.ServerStream, _ *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	return func(srv interface{}, ss grpc.ServerStream, _ *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		wrapped := WrapServerStream(ss)
		wrapped.WrappedContext = contextWithClient(ss.Context(), includeMetadata)
		return handler(srv, wrapped)
	}
}

// contextWithClient attempts to add the peer address to the client.Info from the context. When no
// client.Info exists in the context, one is created.
func contextWithClient(ctx context.Context, includeMetadata bool) context.Context {
	cl := client.FromContext(ctx)
	if p, ok := peer.FromContext(ctx); ok {
		cl.Addr = p.Addr
	}
	if includeMetadata {
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			copiedMD := md.Copy()
			if len(md[client.MetadataHostName]) == 0 && len(md[":authority"]) > 0 {
				copiedMD[client.MetadataHostName] = md[":authority"]
			}
			cl.Metadata = client.NewMetadata(copiedMD)
		}
	}
	return client.NewContext(ctx, cl)
}

func authUnaryServerInterceptor(ctx context.Context, req interface{}, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler, authenticate configauth.AuthenticateFunc) (interface{}, error) {
	headers, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, errMetadataNotFound
	}

	ctx, err := authenticate(ctx, headers)
	if err != nil {
		return nil, err
	}

	return handler(ctx, req)
}

func authStreamServerInterceptor(srv interface{}, stream grpc.ServerStream, _ *grpc.StreamServerInfo, handler grpc.StreamHandler, authenticate configauth.AuthenticateFunc) error {
	ctx := stream.Context()
	headers, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return errMetadataNotFound
	}

	ctx, err := authenticate(ctx, headers)
	if err != nil {
		return err
	}

	wrapped := WrapServerStream(stream)
	wrapped.WrappedContext = ctx
	return handler(srv, wrapped)
}

// WrappedServerStream is a thin wrapper around grpc.ServerStream that allows modifying context.
type WrappedServerStream struct {
	grpc.ServerStream
	// WrappedContext is the wrapper's own Context. You can assign it.
	WrappedContext context.Context
}

// Context returns the wrapper's WrappedContext, overwriting the nested grpc.ServerStream.Context()
func (w *WrappedServerStream) Context() context.Context {
	return w.WrappedContext
}

// WrapServerStream returns a ServerStream that has the ability to overwrite context.
func WrapServerStream(stream grpc.ServerStream) *WrappedServerStream {
	if existing, ok := stream.(*WrappedServerStream); ok {
		return existing
	}
	return &WrappedServerStream{ServerStream: stream, WrappedContext: stream.Context()}
}
