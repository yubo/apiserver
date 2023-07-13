package server

import (
	"crypto/sha256"
	"encoding/base32"
	"fmt"
	"net"
	"net/http"
	"os"
	goruntime "runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/go-openapi/spec"
	"github.com/google/uuid"
	"github.com/yubo/apiserver/components/logs"
	"github.com/yubo/apiserver/components/metrics/prometheus/slis"
	"github.com/yubo/apiserver/pkg/audit"
	"github.com/yubo/apiserver/pkg/authentication/authenticator"
	"github.com/yubo/apiserver/pkg/authentication/authenticatorfactory"
	authenticatorunion "github.com/yubo/apiserver/pkg/authentication/request/union"
	"github.com/yubo/apiserver/pkg/authentication/user"
	"github.com/yubo/apiserver/pkg/authorization/authorizer"
	"github.com/yubo/apiserver/pkg/authorization/authorizerfactory"
	authorizerunion "github.com/yubo/apiserver/pkg/authorization/union"
	"github.com/yubo/apiserver/pkg/filters"
	apirequest "github.com/yubo/apiserver/pkg/request"
	"github.com/yubo/apiserver/pkg/server/dynamiccertificates"
	"github.com/yubo/apiserver/pkg/server/healthz"
	"github.com/yubo/apiserver/pkg/server/routes"
	"github.com/yubo/apiserver/pkg/sessions"
	"github.com/yubo/apiserver/pkg/tracing"
	restclient "github.com/yubo/client-go/rest"
	"github.com/yubo/golib/runtime"
	"github.com/yubo/golib/runtime/serializer"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/crypto/cryptobyte"
	"k8s.io/klog/v2"

	//utilflowcontrol "github.com/yubo/apiserver/pkg/util/flowcontrol"
	"github.com/yubo/golib/util/clock"
	utilnet "github.com/yubo/golib/util/net"
	"github.com/yubo/golib/util/sets"
	utilwaitgroup "github.com/yubo/golib/util/waitgroup"
	"github.com/yubo/golib/version"
)

const (
	// DefaultLegacyAPIPrefix is where the legacy APIs will be located.
	DefaultLegacyAPIPrefix = "/api"

	// APIGroupPrefix is where non-legacy API group will be located.
	APIGroupPrefix = "/apis"

	APIDocsPath = "/apidocs.json"
)

// runtime config
type Config struct {
	// SecureServing is required to serve https
	SecureServing *SecureServingInfo

	// customize
	InsecureServing *DeprecatedInsecureServingInfo

	// Authentication is the configuration for authentication
	Authentication AuthenticationInfo

	// Authorization is the configuration for authorization
	Authorization AuthorizationInfo

	// LoopbackClientConfig is a config for a privileged loopback connection to the API server
	// This is required for proper functioning of the PostStartHooks on a GenericAPIServer
	// TODO: move into SecureServing(WithLoopback) as soon as insecure serving is gone
	LoopbackClientConfig *restclient.Config

	// EgressSelector provides a lookup mechanism for dialing outbound connections.
	// It does so based on a EgressSelectorConfiguration which was read at startup.
	//EgressSelector *egressselector.EgressSelector

	// RuleResolver is required to get the list of rules that apply to a given user
	// in a given namespace
	RuleResolver authorizer.RuleResolver

	// AdmissionControl performs deep inspection of a given request (including content)
	// to set values and determine whether its allowed
	//AdmissionControl      admission.Interface

	CorsAllowedOriginList []string
	HSTSDirectives        []string
	// FlowControl, if not nil, gives priority and fairness to request handling
	//FlowControl utilflowcontrol.Interface
	FlowControl interface{}

	// features
	EnableIndex     bool
	EnableProfiling bool
	DebugSocketPath string
	EnableDiscovery bool
	EnableExpvar    bool
	EnableHealthz   bool

	// Requires generic profiling enabled
	EnableContentionProfiling bool
	EnableMetrics             bool

	DisabledPostStartHooks sets.String
	// done values in this values for this map are ignored.
	PostStartHooks map[string]PostStartHookConfigEntry

	// Version will enable the /version endpoint if non-nil
	Version *version.Info
	// AuditBackend is where audit events are sent to.(option)
	AuditBackend audit.Backend
	// AuditPolicyRuleEvaluator makes the decision of whether and how to audit log a request.
	AuditPolicyRuleEvaluator audit.PolicyRuleEvaluator
	// ExternalAddress is the host name to use for external (public internet) facing URLs (e.g. Swagger)
	// Will default to a value based on secure serving info and available ipv4 IPs.
	ExternalAddress string

	// TracerProvider can provide a tracer, which records spans for distributed tracing.
	TracerProvider trace.TracerProvider

	//===========================================================================
	// Fields you probably don't care about changing
	//===========================================================================

	// BuildHandlerChainFunc allows you to build custom handler chains by decorating the apiHandler.
	BuildHandlerChainFunc func(apiHandler http.Handler, s *Config) (secure http.Handler)
	// NonLongRunningRequestWaitGroup allows you to wait for all chain
	// handlers associated with non long-running requests
	// to complete while the server is shuting down.
	NonLongRunningRequestWaitGroup *utilwaitgroup.SafeWaitGroup
	// WatchRequestWaitGroup allows us to wait for all chain
	// handlers associated with active watch requests to
	// complete while the server is shuting down.
	WatchRequestWaitGroup *utilwaitgroup.RateLimitedSafeWaitGroup

	// DiscoveryAddresses is used to build the IPs pass to discovery. If nil, the ExternalAddress is
	// always reported
	//DiscoveryAddresses discovery.Addresses

	// The default set of healthz checks. There might be more added via AddHealthChecks dynamically.
	HealthzChecks []healthz.HealthChecker
	// The default set of livez checks. There might be more added via AddHealthChecks dynamically.
	LivezChecks []healthz.HealthChecker
	// The default set of readyz-only checks. There might be more added via AddReadyzChecks dynamically.
	ReadyzChecks []healthz.HealthChecker
	// LegacyAPIGroupPrefixes is used to set up URL parsing for authorization and for validating requests
	// to InstallLegacyAPIGroup. New API servers don't generally have legacy groups at all.
	LegacyAPIGroupPrefixes sets.String
	// RequestInfoResolver is used to assign attributes (used by admission and authorization) based on a request URL.
	// Use-cases that are like kubelets may need to customize this.
	RequestInfoResolver apirequest.RequestInfoResolver
	// Serializer is required and provides the interface for serializing and converting objects to and from the wire
	// The default (api.Codecs) usually works fine.
	Serializer runtime.NegotiatedSerializer
	// OpenAPIConfig will be used in generating OpenAPI spec. This is nil by default. Use DefaultOpenAPIConfig for "working" defaults.
	//OpenAPIConfig *openapicommon.Config
	// OpenAPIV3Config will be used in generating OpenAPI V3 spec. This is nil by default. Use DefaultOpenAPIV3Config for "working" defaults.
	//OpenAPIV3Config *openapicommon.Config
	// SkipOpenAPIInstallation avoids installing the OpenAPI handler if set to true.
	SkipOpenAPIInstallation bool

	// RESTOptionsGetter is used to construct RESTStorage types via the generic registry.
	//RESTOptionsGetter genericregistry.RESTOptionsGetter

	// If specified, all requests except those which match the LongRunningFunc predicate will timeout
	// after this duration.
	RequestTimeout time.Duration
	// If specified, long running requests such as watch will be allocated a random timeout between this value, and
	// twice this value.  Note that it is up to the request handlers to ignore or honor this timeout. In seconds.
	MinRequestTimeout int

	// This represents the maximum amount of time it should take for apiserver to complete its startup
	// sequence and become healthy. From apiserver's start time to when this amount of time has
	// elapsed, /livez will assume that unfinished post-start hooks will complete successfully and
	// therefore return true.
	LivezGracePeriod time.Duration
	// ShutdownDelayDuration allows to block shutdown for some time, e.g. until endpoints pointing to this API server
	// have converged on all node. During this time, the API server keeps serving, /healthz will return 200,
	// but /readyz will return failure.
	ShutdownDelayDuration time.Duration

	// The limit on the total size increase all "copy" operations in a json
	// patch may cause.
	// This affects all places that applies json patch in the binary.
	JSONPatchMaxCopyBytes int64
	// The limit on the request size that would be accepted and decoded in a write request
	// 0 means no limit.
	MaxRequestBodyBytes int64
	// MaxRequestsInFlight is the maximum number of parallel non-long-running requests. Every further
	// request has to wait. Applies only to non-mutating requests.
	MaxRequestsInFlight int
	// MaxMutatingRequestsInFlight is the maximum number of parallel mutating requests. Every further
	// request has to wait.
	MaxMutatingRequestsInFlight int
	// Predicate which is true for paths of long-running http requests
	LongRunningFunc apirequest.LongRunningRequestCheck

	// GoawayChance is the probability that send a GOAWAY to HTTP/2 clients. When client received
	// GOAWAY, the in-flight requests will not be affected and new requests will use
	// a new TCP connection to triggering re-balancing to another server behind the load balance.
	// Default to 0, means never send GOAWAY. Max is 0.02 to prevent break the apiserver.
	GoawayChance float64

	// MergedResourceConfig indicates which groupVersion enabled and its resources enabled/disabled.
	// This is composed of genericapiserver defaultAPIResourceConfig and those parsed from flags.
	// If not specify any in flags, then genericapiserver will only enable defaultAPIResourceConfig.
	//MergedResourceConfig *serverstore.ResourceConfig

	// lifecycleSignals provides access to the various signals
	// that happen during lifecycle of the apiserver.
	// it's intentionally marked private as it should never be overridden.
	lifecycleSignals lifecycleSignals

	// StorageObjectCountTracker is used to keep track of the total number of objects
	// in the storage per resource, so we can estimate width of incoming requests.
	//StorageObjectCountTracker flowcontrolrequest.StorageObjectCountTracker

	// ShutdownSendRetryAfter dictates when to initiate shutdown of the HTTP
	// Server during the graceful termination of the apiserver. If true, we wait
	// for non longrunning requests in flight to be drained and then initiate a
	// shutdown of the HTTP Server. If false, we initiate a shutdown of the HTTP
	// Server as soon as ShutdownDelayDuration has elapsed.
	// If enabled, after ShutdownDelayDuration elapses, any incoming request is
	// rejected with a 429 status code and a 'Retry-After' response.
	ShutdownSendRetryAfter bool

	//===========================================================================
	// values below here are targets for removal
	//===========================================================================

	// PublicAddress is the IP address where members of the cluster (kubelet,
	// kube-proxy, services, etc.) can reach the GenericAPIServer.
	// If nil or 0.0.0.0, the host's default interface will be used.
	PublicAddress net.IP

	// EquivalentResourceRegistry provides information about resources equivalent to a given resource,
	// and the kind associated with a given resource. As resources are installed, they are registered here.
	//EquivalentResourceRegistry runtime.EquivalentResourceRegistry

	// APIServerID is the ID of this API server
	APIServerID string

	// StorageVersionManager holds the storage versions of the API resources installed by this server.
	//StorageVersionManager storageversion.Manager

	// AggregatedDiscoveryGroupManager serves /apis in an aggregated form.
	//AggregatedDiscoveryGroupManager discoveryendpoint.ResourceManager

	// ShutdownWatchTerminationGracePeriod, if set to a positive value,
	// is the maximum duration the apiserver will wait for all active
	// watch request(s) to drain.
	// Once this grace period elapses, the apiserver will no longer
	// wait for any active watch request(s) in flight to drain, it will
	// proceed to the next step in the graceful server shutdown process.
	// If set to a positive value, the apiserver will keep track of the
	// number of active watch request(s) in flight and during shutdown
	// it will wait, at most, for the specified duration and allow these
	// active watch requests to drain with some rate limiting in effect.
	// The default is zero, which implies the apiserver will not keep
	// track of active watch request(s) in flight and will not wait
	// for them to drain, this maintains backward compatibility.
	// This grace period is orthogonal to other grace periods, and
	// it is not overridden by any other grace period.
	ShutdownWatchTerminationGracePeriod time.Duration

	//===========================================================================
	// values below here are targets for customize

	//===========================================================================
	// Handler holds the handlers being used by this API server
	//Handler *APIServerHandler
	// ListedPathProvider is a lister which provides the set of paths to show at /
	//ListedPathProvider routes.ListedPathProvider

	SecuritySchemes []*spec.SecurityScheme
}

type SecureServingInfo struct {
	// Listener is the secure server network listener.
	Listener net.Listener

	// Cert is the main server cert which is used if SNI does not match. Cert must be non-nil and is
	// allowed to be in SNICerts.
	Cert dynamiccertificates.CertKeyContentProvider

	// SNICerts are the TLS certificates used for SNI.
	SNICerts []dynamiccertificates.SNICertKeyContentProvider

	// ClientCA is the certificate bundle for all the signers that you'll recognize for incoming client certificates
	ClientCA dynamiccertificates.CAContentProvider

	// MinTLSVersion optionally overrides the minimum TLS version supported.
	// Values are from tls package constants (https://golang.org/pkg/crypto/tls/#pkg-constants).
	MinTLSVersion uint16

	// CipherSuites optionally overrides the list of allowed cipher suites for the server.
	// Values are from tls package constants (https://golang.org/pkg/crypto/tls/#pkg-constants).
	CipherSuites []uint16

	// HTTP2MaxStreamsPerConnection is the limit that the api server imposes on each client.
	// A value of zero means to use the default provided by golang's HTTP/2 support.
	HTTP2MaxStreamsPerConnection int

	// DisableHTTP2 indicates that http2 should not be enabled.
	DisableHTTP2 bool
}

type AuthenticationInfo struct {
	// APIAudiences is a list of identifier that the API identifies as. This is
	// used by some authenticators to validate audience bound credentials.
	APIAudiences authenticator.Audiences
	// Authenticator determines which subject is making the request
	Authenticator authenticator.Request

	// plugin authentication.requestheader
	RequestHeaderConfig *authenticatorfactory.RequestHeaderConfig

	// customize
	Anonymous bool
}

type AuthorizationInfo struct {
	// Authorizer determines whether the subject is allowed to make the request based only
	// on the RequestURI
	Authorizer authorizer.Authorizer

	// customize
	Modes sets.String
}

// NewConfig returns a Config struct with the default values
func NewConfig(codecs serializer.CodecFactory) *Config {
	defaultHealthChecks := []healthz.HealthChecker{healthz.PingHealthz, healthz.LogHealthz}
	var id string
	hostname, err := os.Hostname()
	if err != nil {
		klog.Fatalf("error getting hostname for apiserver identity: %v", err)
	}

	// Since the hash needs to be unique across each kube-apiserver and aggregated apiservers,
	// the hash used for the identity should include both the hostname and the identity value.
	// TODO: receive the identity value as a parameter once the apiserver identity lease controller
	// post start hook is moved to generic apiserver.
	b := cryptobyte.NewBuilder(nil)
	b.AddUint16LengthPrefixed(func(b *cryptobyte.Builder) {
		b.AddBytes([]byte(hostname))
	})
	b.AddUint16LengthPrefixed(func(b *cryptobyte.Builder) {
		b.AddBytes([]byte("kube-apiserver"))
	})
	hashData, err := b.Bytes()
	if err != nil {
		klog.Fatalf("error building hash data for apiserver identity: %v", err)
	}

	hash := sha256.Sum256(hashData)
	id = "apiserver-" + strings.ToLower(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(hash[:16]))
	lifecycleSignals := newLifecycleSignals()

	return &Config{
		Serializer:                     codecs,
		BuildHandlerChainFunc:          DefaultBuildHandlerChain,
		NonLongRunningRequestWaitGroup: new(utilwaitgroup.SafeWaitGroup),
		WatchRequestWaitGroup:          &utilwaitgroup.RateLimitedSafeWaitGroup{},
		LegacyAPIGroupPrefixes:         sets.NewString(DefaultLegacyAPIPrefix),
		DisabledPostStartHooks:         sets.NewString(),
		PostStartHooks:                 map[string]PostStartHookConfigEntry{},
		HealthzChecks:                  append([]healthz.HealthChecker{}, defaultHealthChecks...),
		ReadyzChecks:                   append([]healthz.HealthChecker{}, defaultHealthChecks...),
		LivezChecks:                    append([]healthz.HealthChecker{}, defaultHealthChecks...),
		EnableIndex:                    true,
		EnableDiscovery:                true,
		EnableProfiling:                true,
		DebugSocketPath:                "",
		EnableMetrics:                  true,
		MaxRequestsInFlight:            400,
		MaxMutatingRequestsInFlight:    200,
		RequestTimeout:                 time.Duration(60) * time.Second,
		MinRequestTimeout:              1800,
		LivezGracePeriod:               time.Duration(0),
		ShutdownDelayDuration:          time.Duration(0),
		// 1.5MB is the default client request size in bytes
		// the etcd server should accept. See
		// https://github.com/etcd-io/etcd/blob/release-3.4/embed/config.go#L56.
		// A request body might be encoded in json, and is converted to
		// proto when persisted in etcd, so we allow 2x as the largest size
		// increase the "copy" operations in a json patch may cause.
		JSONPatchMaxCopyBytes: int64(3 * 1024 * 1024),
		// 1.5MB is the recommended client request size in byte
		// the etcd server should accept. See
		// https://github.com/etcd-io/etcd/blob/release-3.4/embed/config.go#L56.
		// A request body might be encoded in json, and is converted to
		// proto when persisted in etcd, so we allow 2x as the largest request
		// body size to be accepted and decoded in a write request.
		// If this constant is changed, DefaultMaxRequestSizeBytes in k8s.io/apiserver/pkg/cel/limits.go
		// should be changed to reflect the new value, if the two haven't
		// been wired together already somehow.
		MaxRequestBodyBytes: int64(3 * 1024 * 1024),

		// Default to treating watch as a long-running operation
		// Generic API servers have no inherent long-running subresources
		LongRunningFunc:  filters.BasicLongRunningRequestCheck(sets.NewString("watch"), sets.NewString()),
		lifecycleSignals: lifecycleSignals,
		//StorageObjectCountTracker:           flowcontrolrequest.NewStorageObjectCountTracker(),
		ShutdownWatchTerminationGracePeriod: time.Duration(0),

		APIServerID: id,
		//StorageVersionManager: storageversion.NewDefaultManager(),
		TracerProvider: tracing.NewNoopTracerProvider(),
	}
}

func (c *AuthenticationInfo) ApplyClientCert(clientCA dynamiccertificates.CAContentProvider, servingInfo *SecureServingInfo) error {
	if servingInfo == nil {
		return nil
	}
	if clientCA == nil {
		return nil
	}
	if servingInfo.ClientCA == nil {
		servingInfo.ClientCA = clientCA
		return nil
	}

	servingInfo.ClientCA = dynamiccertificates.NewUnionCAContentProvider(servingInfo.ClientCA, clientCA)
	return nil
}

type completedConfig struct {
	*Config

	//===========================================================================
	// values below here are filled in during completion
	//===========================================================================

	// SharedInformerFactory provides shared informers for resources
	//SharedInformerFactory informers.SharedInformerFactory
}

type CompletedConfig struct {
	// Embed a private pointer that cannot be instantiated outside of this package.
	*completedConfig
}

// AddHealthChecks adds a health check to our config to be exposed by the health endpoints
// of our configured apiserver. We should prefer this to adding healthChecks directly to
// the config unless we explicitly want to add a healthcheck only to a specific health endpoint.
func (c *Config) AddHealthChecks(healthChecks ...healthz.HealthChecker) {
	c.HealthzChecks = append(c.HealthzChecks, healthChecks...)
	c.LivezChecks = append(c.LivezChecks, healthChecks...)
	c.ReadyzChecks = append(c.ReadyzChecks, healthChecks...)
}

// AddReadyzChecks adds a health check to our config to be exposed by the readyz endpoint
// of our configured apiserver.
func (c *Config) AddReadyzChecks(healthChecks ...healthz.HealthChecker) {
	c.ReadyzChecks = append(c.ReadyzChecks, healthChecks...)
}

// DrainedNotify returns a lifecycle signal of genericapiserver already drained while shutting down.
func (c *Config) DrainedNotify() <-chan struct{} {
	return c.lifecycleSignals.InFlightRequestsDrained.Signaled()
}

// Complete fills in any fields not set that are required to have valid data and can be derived
// from other fields. If you're going to `ApplyOptions`, do that first. It's mutating the receiver.
func (c *Config) Complete() CompletedConfig {
	if len(c.ExternalAddress) == 0 && c.PublicAddress != nil {
		c.ExternalAddress = c.PublicAddress.String()
	}

	// if there is no port, and we listen on one securely, use that one
	if _, _, err := net.SplitHostPort(c.ExternalAddress); err != nil {
		var port int
		if c.SecureServing != nil {
			_, port, err = c.SecureServing.HostPort()
		} else if c.InsecureServing != nil {
			_, port, err = c.InsecureServing.HostPort()
		} else {
			klog.Fatalf("cannot derive external address port without listening on a secure/insecure port.")
		}

		if err != nil {
			klog.Fatalf("cannot derive external address from the secure/insecure port: %v", err)
		}
		c.ExternalAddress = net.JoinHostPort(c.ExternalAddress, strconv.Itoa(port))
	}

	//completeOpenAPI(c.OpenAPIConfig, c.Version)
	//completeOpenAPI(c.OpenAPIV3Config, c.Version)

	//if c.DiscoveryAddresses == nil {
	//	c.DiscoveryAddresses = discovery.DefaultAddresses{DefaultAddress: c.ExternalAddress}
	//}

	//AuthorizeClientBearerToken(c.LoopbackClientConfig, &c.Authentication, &c.Authorization)

	if c.RequestInfoResolver == nil {
		c.RequestInfoResolver = NewRequestInfoResolver(c)
	}

	//if c.EquivalentResourceRegistry == nil {
	//	if c.RESTOptionsGetter == nil {
	//		c.EquivalentResourceRegistry = runtime.NewEquivalentResourceRegistry()
	//	} else {
	//		c.EquivalentResourceRegistry = runtime.NewEquivalentResourceRegistryWithIdentity(func(groupResource schema.GroupResource) string {
	//			// use the storage prefix as the key if possible
	//			if opts, err := c.RESTOptionsGetter.GetRESTOptions(groupResource); err == nil {
	//				return opts.ResourcePrefix
	//			}
	//			// otherwise return "" to use the default key (parent GV name)
	//			return ""
	//		})
	//	}
	//}

	return CompletedConfig{&completedConfig{c}}
}

// New creates a new server which logically combines the handling chain with the passed server.
// name is used to differentiate for logging. The handler chain in particular can be difficult as it starts delegating.
// delegationTarget may not be nil.
func (c completedConfig) New(name string, delegationTarget DelegationTarget) (*GenericAPIServer, error) {
	if c.Serializer == nil {
		return nil, fmt.Errorf("Genericapiserver.New() called with config.Serializer == nil")
	}
	if c.LoopbackClientConfig == nil {
		return nil, fmt.Errorf("Genericapiserver.New() called with config.LoopbackClientConfig == nil")
	}
	//if c.EquivalentResourceRegistry == nil {
	//	return nil, fmt.Errorf("Genericapiserver.New() called with config.EquivalentResourceRegistry == nil")
	//}

	handlerChainBuilder := func(handler http.Handler) http.Handler {
		return c.BuildHandlerChainFunc(handler, c.Config)
	}

	var debugSocket *routes.DebugSocket
	if c.DebugSocketPath != "" {
		debugSocket = routes.NewDebugSocket(c.DebugSocketPath)
	}

	apiServerHandler := NewAPIServerHandler(name, c.Serializer, handlerChainBuilder, delegationTarget.UnprotectedHandler())

	s := &GenericAPIServer{
		//discoveryAddresses:             c.DiscoveryAddresses,
		LoopbackClientConfig:   c.LoopbackClientConfig,
		legacyAPIGroupPrefixes: c.LegacyAPIGroupPrefixes,
		//admissionControl:               c.AdmissionControl,
		Serializer:   c.Serializer,
		AuditBackend: c.AuditBackend,
		Authorizer:   c.Authorization.Authorizer, // by init2

		delegationTarget: delegationTarget,
		//EquivalentResourceRegistry:     c.EquivalentResourceRegistry,
		NonLongRunningRequestWaitGroup: c.NonLongRunningRequestWaitGroup,
		WatchRequestWaitGroup:          c.WatchRequestWaitGroup,
		Handler:                        apiServerHandler,
		UnprotectedDebugSocket:         debugSocket,

		listedPathProvider: apiServerHandler,

		minRequestTimeout:                   time.Duration(c.MinRequestTimeout) * time.Second,
		ShutdownTimeout:                     c.RequestTimeout,
		ShutdownDelayDuration:               c.ShutdownDelayDuration,
		ShutdownWatchTerminationGracePeriod: c.ShutdownWatchTerminationGracePeriod,
		SecureServingInfo:                   c.SecureServing,
		InsecureServingInfo:                 c.InsecureServing,
		ExternalAddress:                     c.ExternalAddress,

		//openAPIConfig:           c.OpenAPIConfig,
		//openAPIV3Config:         c.OpenAPIV3Config,
		skipOpenAPIInstallation: c.SkipOpenAPIInstallation,

		postStartHooks:         map[string]postStartHookEntry{},
		preShutdownHooks:       map[string]preShutdownHookEntry{},
		disabledPostStartHooks: c.DisabledPostStartHooks,

		healthzChecks:    c.HealthzChecks,
		livezChecks:      c.LivezChecks,
		readyzChecks:     c.ReadyzChecks,
		livezGracePeriod: c.LivezGracePeriod,

		//DiscoveryGroupManager: discovery.NewRootAPIsHandler(c.DiscoveryAddresses, c.Serializer),

		maxRequestBodyBytes: c.MaxRequestBodyBytes,
		livezClock:          clock.RealClock{},

		lifecycleSignals:       c.lifecycleSignals,
		ShutdownSendRetryAfter: c.ShutdownSendRetryAfter,

		APIServerID: c.APIServerID,
		//StorageVersionManager: c.StorageVersionManager,

		Version: c.Version,

		muxAndDiscoveryCompleteSignals: map[string]<-chan struct{}{},
	}

	//if utilfeature.DefaultFeatureGate.Enabled(genericfeatures.AggregatedDiscoveryEndpoint) {
	//	manager := c.AggregatedDiscoveryGroupManager
	//	if manager == nil {
	//		manager = discoveryendpoint.NewResourceManager("apis")
	//	}
	//	s.AggregatedDiscoveryGroupManager = manager
	//	s.AggregatedLegacyDiscoveryGroupManager = discoveryendpoint.NewResourceManager("api")
	//}
	for {
		if c.JSONPatchMaxCopyBytes <= 0 {
			break
		}
		existing := atomic.LoadInt64(&jsonpatch.AccumulatedCopySizeLimit)
		if existing > 0 && existing < c.JSONPatchMaxCopyBytes {
			break
		}
		if atomic.CompareAndSwapInt64(&jsonpatch.AccumulatedCopySizeLimit, existing, c.JSONPatchMaxCopyBytes) {
			break
		}
	}

	// first add poststarthooks from delegated targets
	for k, v := range delegationTarget.PostStartHooks() {
		s.postStartHooks[k] = v
	}

	for k, v := range delegationTarget.PreShutdownHooks() {
		s.preShutdownHooks[k] = v
	}

	// add poststarthooks that were preconfigured.  Using the add method will give us an error if the same name has already been registered.
	//for name, preconfiguredPostStartHook := range c.PostStartHooks {
	//	if err := s.AddPostStartHook(name, preconfiguredPostStartHook.hook); err != nil {
	//		return nil, err
	//	}
	//}

	// register mux signals from the delegated server
	for k, v := range delegationTarget.MuxAndDiscoveryCompleteSignals() {
		if err := s.RegisterMuxAndDiscoveryCompleteSignal(k, v); err != nil {
			return nil, err
		}
	}

	//genericApiServerHookName := "generic-apiserver-start-informers"
	//if c.SharedInformerFactory != nil {
	//	if !s.isPostStartHookRegistered(genericApiServerHookName) {
	//		err := s.AddPostStartHook(genericApiServerHookName, func(context PostStartHookContext) error {
	//			c.SharedInformerFactory.Start(context.StopCh)
	//			return nil
	//		})
	//		if err != nil {
	//			return nil, err
	//		}
	//	}
	//	// TODO: Once we get rid of /healthz consider changing this to post-start-hook.
	//	err := s.AddReadyzChecks(healthz.NewInformerSyncHealthz(c.SharedInformerFactory))
	//	if err != nil {
	//		return nil, err
	//	}
	//}

	const priorityAndFairnessConfigConsumerHookName = "priority-and-fairness-config-consumer"
	if s.isPostStartHookRegistered(priorityAndFairnessConfigConsumerHookName) {
	} else if c.FlowControl != nil {
		//	err := s.AddPostStartHook(priorityAndFairnessConfigConsumerHookName, func(context PostStartHookContext) error {
		//		go c.FlowControl.Run(context.StopCh)
		//		return nil
		//	})
		//	if err != nil {
		//		return nil, err
		//	}
		//	// TODO(yue9944882): plumb pre-shutdown-hook for request-management system?
	} else {
		klog.V(3).Infof("Not requested to run hook %s", priorityAndFairnessConfigConsumerHookName)
	}

	// Add PostStartHooks for maintaining the watermarks for the Priority-and-Fairness and the Max-in-Flight filters.
	if c.FlowControl != nil {
		//const priorityAndFairnessFilterHookName = "priority-and-fairness-filter"
		//if !s.isPostStartHookRegistered(priorityAndFairnessFilterHookName) {
		//	err := s.AddPostStartHook(priorityAndFairnessFilterHookName, func(context PostStartHookContext) error {
		//		filters.StartPriorityAndFairnessWatermarkMaintenance(context.StopCh)
		//		return nil
		//	})
		//	if err != nil {
		//		return nil, err
		//	}
		//}
	} else {
		const maxInFlightFilterHookName = "max-in-flight-filter"
		if !s.isPostStartHookRegistered(maxInFlightFilterHookName) {
			err := s.AddPostStartHook(maxInFlightFilterHookName, func(context PostStartHookContext) error {
				filters.StartMaxInFlightWatermarkMaintenance(context.StopCh)
				return nil
			})
			if err != nil {
				return nil, err
			}
		}
	}

	// Add PostStartHook for maintenaing the object count tracker.
	//if c.StorageObjectCountTracker != nil {
	//	const storageObjectCountTrackerHookName = "storage-object-count-tracker-hook"
	//	if !s.isPostStartHookRegistered(storageObjectCountTrackerHookName) {
	//		if err := s.AddPostStartHook(storageObjectCountTrackerHookName, func(context PostStartHookContext) error {
	//			go c.StorageObjectCountTracker.RunUntil(context.StopCh)
	//			return nil
	//		}); err != nil {
	//			return nil, err
	//		}
	//	}
	//}

	for _, delegateCheck := range delegationTarget.HealthzChecks() {
		skip := false
		for _, existingCheck := range c.HealthzChecks {
			if existingCheck.Name() == delegateCheck.Name() {
				skip = true
				break
			}
		}
		if skip {
			continue
		}
		s.AddHealthChecks(delegateCheck)
	}

	// shutdown with itself
	//s.RegisterDestroyFunc(func() {
	//	if err := c.Config.TracerProvider.Shutdown(context.Background()); err != nil {
	//		klog.Errorf("failed to shut down tracer provider: %v", err)
	//	}
	//})

	s.listedPathProvider = routes.ListedPathProviders{s.listedPathProvider, delegationTarget}

	installAPI(s, c.Config)

	// use the UnprotectedHandler from the delegation target to ensure that we don't attempt to double authenticator, authorize,
	// or some other part of the filter chain in delegation cases.
	if delegationTarget.UnprotectedHandler() == nil && c.EnableIndex {
		s.Handler.NonGoRestfulMux.NotFoundHandler(routes.IndexLister{
			StatusCode:   http.StatusNotFound,
			PathProvider: s.listedPathProvider,
		})
	}

	return s, nil
}

func DefaultBuildHandlerChain(apiHandler http.Handler, c *Config) http.Handler {
	handler := filters.TrackCompleted(apiHandler)
	handler = filters.WithAuthorization(handler, c.Authorization.Authorizer, c.Serializer)
	handler = filters.TrackStarted(handler, c.TracerProvider, "authorization")

	//if c.FlowControl != nil {
	//	workEstimatorCfg := flowcontrolrequest.DefaultWorkEstimatorConfig()
	//	requestWorkEstimator := flowcontrolrequest.NewWorkEstimator(
	//		c.StorageObjectCountTracker.Get, c.FlowControl.GetInterestedWatchCount, workEstimatorCfg)
	//	handler = filterlatency.TrackCompleted(handler)
	//	handler = genericfilters.WithPriorityAndFairness(handler, c.LongRunningFunc, c.FlowControl, requestWorkEstimator)
	//	handler = filterlatency.TrackStarted(handler, c.TracerProvider, "priorityandfairness")
	//} else {
	handler = filters.WithMaxInFlightLimit(handler, c.MaxRequestsInFlight, c.MaxMutatingRequestsInFlight, c.LongRunningFunc)
	//}

	handler = filters.TrackCompleted(handler)
	handler = filters.WithImpersonation(handler, c.Authorization.Authorizer, c.Serializer)
	handler = filters.TrackStarted(handler, c.TracerProvider, "impersonation")

	handler = filters.TrackCompleted(handler)
	handler = filters.WithAudit(handler, c.AuditBackend, c.AuditPolicyRuleEvaluator, c.LongRunningFunc)
	handler = filters.TrackStarted(handler, c.TracerProvider, "audit")

	failedHandler := filters.Unauthorized(c.Serializer)
	failedHandler = filters.WithFailedAuthenticationAudit(failedHandler, c.AuditBackend, c.AuditPolicyRuleEvaluator)

	failedHandler = filters.TrackCompleted(failedHandler)
	handler = filters.TrackCompleted(handler)
	handler = filters.WithAuthentication(handler, c.Authentication.Authenticator, failedHandler, c.Authentication.APIAudiences, c.Authentication.RequestHeaderConfig)
	handler = filters.TrackStarted(handler, c.TracerProvider, "authentication")

	handler = sessions.WithSessions(handler)

	handler = filters.WithCORS(handler, c.CorsAllowedOriginList, nil, nil, nil, "true")

	// WithTimeoutForNonLongRunningRequests will call the rest of the request handling in a go-routine with the
	// context with deadline. The go-routine can keep running, while the timeout logic will return a timeout to the client.
	handler = filters.WithTimeoutForNonLongRunningRequests(handler, c.LongRunningFunc)

	handler = filters.WithRequestDeadline(handler, c.AuditBackend, c.AuditPolicyRuleEvaluator, c.LongRunningFunc, c.Serializer, c.RequestTimeout)
	handler = filters.WithWaitGroup(handler, c.LongRunningFunc, c.NonLongRunningRequestWaitGroup)
	if c.ShutdownWatchTerminationGracePeriod > 0 {
		handler = filters.WithWatchTerminationDuringShutdown(handler, c.lifecycleSignals, c.WatchRequestWaitGroup)
	}
	if c.SecureServing != nil && !c.SecureServing.DisableHTTP2 && c.GoawayChance > 0 {
		handler = filters.WithProbabilisticGoaway(handler, c.GoawayChance)
	}
	handler = filters.WithWarningRecorder(handler)
	handler = filters.WithCacheControl(handler)
	handler = filters.WithHSTS(handler, c.HSTSDirectives)
	if c.ShutdownSendRetryAfter {
		handler = filters.WithRetryAfter(handler, c.lifecycleSignals.NotAcceptingNewRequest.Signaled())
	}

	handler = filters.WithHTTPLogging(handler)
	handler = tracing.WithTracing(handler, tracing.WithTracerProvider(c.TracerProvider))
	handler = filters.WithLatencyTrackers(handler)
	handler = filters.WithRequestInfo(handler, c.RequestInfoResolver)
	handler = filters.WithRequestReceivedTimestamp(handler)
	handler = filters.WithMuxAndDiscoveryComplete(handler, c.lifecycleSignals.MuxAndDiscoveryComplete.Signaled())
	handler = filters.WithHttpDump(handler)
	handler = filters.WithPanicRecovery(handler, c.RequestInfoResolver)
	handler = filters.WithAuditInit(handler)
	return handler
}

func installAPI(s *GenericAPIServer, c *Config) {
	if c.EnableIndex {
		routes.Index{}.Install(s.listedPathProvider, s.Handler.NonGoRestfulMux)
	}
	if c.EnableProfiling {
		routes.Profiling{}.Install(s.Handler.NonGoRestfulMux)
		if c.EnableContentionProfiling {
			goruntime.SetBlockProfileRate(1)
		}
		// so far, only logging related endpoints are considered valid to add for these debug flags.
		routes.DebugFlags{}.Install(s.Handler.NonGoRestfulMux, "v", routes.StringFlagPutHandler(logs.GlogSetter))
	}
	if s.UnprotectedDebugSocket != nil {
		s.UnprotectedDebugSocket.InstallProfiling()
		s.UnprotectedDebugSocket.InstallDebugFlag("v", routes.StringFlagPutHandler(logs.GlogSetter))
		if c.EnableContentionProfiling {
			goruntime.SetBlockProfileRate(1)
		}
	}

	if c.EnableMetrics {
		if c.EnableProfiling {
			routes.MetricsWithReset{}.Install(s.Handler.NonGoRestfulMux)
			//if utilfeature.DefaultFeatureGate.Enabled(features.ComponentSLIs) {
			slis.SLIMetricsWithReset{}.Install(s.Handler.NonGoRestfulMux)
			//}
		} else {
			routes.DefaultMetrics{}.Install(s.Handler.NonGoRestfulMux)
			//if utilfeature.DefaultFeatureGate.Enabled(features.ComponentSLIs) {
			slis.SLIMetrics{}.Install(s.Handler.NonGoRestfulMux)
			//}
		}
	}

	routes.Version{Version: c.Version}.Install(s.Handler.GoRestfulContainer)

	//if c.EnableDiscovery {
	//	if utilfeature.DefaultFeatureGate.Enabled(genericfeatures.AggregatedDiscoveryEndpoint) {
	//		wrapped := discoveryendpoint.WrapAggregatedDiscoveryToHandler(s.DiscoveryGroupManager, s.AggregatedDiscoveryGroupManager)
	//		s.Handler.GoRestfulContainer.Add(wrapped.GenerateWebService("/apis", metav1.APIGroupList{}))
	//	} else {
	//		s.Handler.GoRestfulContainer.Add(s.DiscoveryGroupManager.WebService())
	//	}
	//}
	//if c.FlowControl != nil && utilfeature.DefaultFeatureGate.Enabled(genericfeatures.APIPriorityAndFairness) {
	//	c.FlowControl.Install(s.Handler.NonGoRestfulMux)
	//}
}

func NewRequestInfoResolver(c *Config) *apirequest.RequestInfoFactory {
	apiPrefixes := sets.NewString(strings.Trim(APIGroupPrefix, "/")) // all possible API prefixes
	legacyAPIPrefixes := sets.String{}                               // APIPrefixes that won't have groups (legacy)
	for legacyAPIPrefix := range c.LegacyAPIGroupPrefixes {
		apiPrefixes.Insert(strings.Trim(legacyAPIPrefix, "/"))
		legacyAPIPrefixes.Insert(strings.Trim(legacyAPIPrefix, "/"))
	}

	return &apirequest.RequestInfoFactory{
		APIPrefixes:          apiPrefixes,
		GrouplessAPIPrefixes: legacyAPIPrefixes,
	}
}

func (s *SecureServingInfo) HostPort() (string, int, error) {
	if s == nil || s.Listener == nil {
		return "", 0, fmt.Errorf("no listener found")
	}
	addr := s.Listener.Addr().String()
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return "", 0, fmt.Errorf("failed to get port from listener address %q: %v", addr, err)
	}
	port, err := utilnet.ParsePort(portStr, true)
	if err != nil {
		return "", 0, fmt.Errorf("invalid non-numeric port %q", portStr)
	}
	return host, port, nil
}

// AuthorizeClientBearerToken wraps the authenticator and authorizer in loopback authentication logic
// if the loopback client config is specified AND it has a bearer token. Note that if either authn or
// authz is nil, this function won't add a token authenticator or authorizer.
func AuthorizeClientBearerToken(loopback *restclient.Config, authn *AuthenticationInfo, authz *AuthorizationInfo) {
	if loopback == nil || len(loopback.BearerToken) == 0 {
		return
	}
	if authn == nil || authz == nil {
		// prevent nil pointer panic
		return
	}
	if authn.Authenticator == nil || authz.Authorizer == nil {
		// authenticator or authorizer might be nil if we want to bypass authz/authn
		// and we also do nothing in this case.
		return
	}

	privilegedLoopbackToken := loopback.BearerToken
	var uid = uuid.New().String()
	tokens := make(map[string]*user.DefaultInfo)
	tokens[privilegedLoopbackToken] = &user.DefaultInfo{
		Name:   user.APIServerUser,
		UID:    uid,
		Groups: []string{user.SystemPrivilegedGroup},
	}

	tokenAuthenticator := authenticatorfactory.NewFromTokens(tokens)
	authn.Authenticator = authenticatorunion.New(tokenAuthenticator, authn.Authenticator)

	tokenAuthorizer := authorizerfactory.NewPrivilegedGroups(user.SystemPrivilegedGroup)
	authz.Authorizer = authorizerunion.New(tokenAuthorizer, authz.Authorizer)
}
