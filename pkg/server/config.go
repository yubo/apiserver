package server

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/emicklei/go-restful/v3"
	"github.com/google/uuid"
	"github.com/yubo/apiserver/pkg/audit"
	"github.com/yubo/apiserver/pkg/authentication/authenticator"
	"github.com/yubo/apiserver/pkg/authentication/authenticatorfactory"
	authenticatorunion "github.com/yubo/apiserver/pkg/authentication/request/union"
	"github.com/yubo/apiserver/pkg/authentication/user"
	"github.com/yubo/apiserver/pkg/authorization/authorizer"
	"github.com/yubo/apiserver/pkg/authorization/authorizerfactory"
	authorizerunion "github.com/yubo/apiserver/pkg/authorization/union"
	"github.com/yubo/apiserver/pkg/dynamiccertificates"
	"github.com/yubo/apiserver/pkg/filters"
	apirequest "github.com/yubo/apiserver/pkg/request"
	"github.com/yubo/apiserver/pkg/rest"
	"github.com/yubo/apiserver/pkg/scheme"
	"github.com/yubo/apiserver/pkg/server/healthz"
	"github.com/yubo/apiserver/pkg/server/routes"
	"github.com/yubo/apiserver/pkg/sessions"
	"github.com/yubo/apiserver/pkg/tracing"
	restclient "github.com/yubo/client-go/rest"
	"github.com/yubo/golib/runtime"
	"go.opentelemetry.io/otel/trace"

	//utilflowcontrol "github.com/yubo/apiserver/pkg/util/flowcontrol"
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

	InsecureServing *DeprecatedInsecureServingInfo

	// Authentication is the configuration for authentication
	Authentication *AuthenticationInfo
	// plugin authentication.requestheader
	RequestHeaderConfig *authenticatorfactory.RequestHeaderConfig

	// Authorization is the configuration for authorization
	Authorization *AuthorizationInfo

	// deprecated
	//Session session.SessionManager

	// LoopbackClientConfig is a config for a privileged loopback connection to the API server
	// This is required for proper functioning of the PostStartHooks on a GenericAPIServer
	// TODO: move into SecureServing(WithLoopback) as soon as insecure serving is gone
	LoopbackClientConfig *restclient.Config

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

	// BuildHandlerChainFunc allows you to build custom handler chains by decorating the apiHandler.
	BuildHandlerChainFunc func(apiHandler http.Handler, s *Config) (secure http.Handler)
	// HandlerChainWaitGroup allows you to wait for all chain handlers exit after the server shutdown.
	HandlerChainWaitGroup *utilwaitgroup.SafeWaitGroup

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

	// Predicate which is true for paths of long-running http requests
	LongRunningFunc apirequest.LongRunningRequestCheck

	// ApiServerID is the ID of this API server
	ApiServerID string

	CorsAllowedOriginList []string
	HSTSDirectives        []string
	// FlowControl, if not nil, gives priority and fairness to request handling
	//FlowControl           utilflowcontrol.Interface
	RequestTimeout        time.Duration
	ShutdownTimeout       time.Duration
	ShutdownDelayDuration time.Duration

	// Handler holds the handlers being used by this API server
	Handler *APIServerHandler
	// ListedPathProvider is a lister which provides the set of paths to show at /
	ListedPathProvider routes.ListedPathProvider

	EnableOpenAPI           bool
	KeepAuthorizationHeader bool
	SecuritySchemes         []rest.SchemeConfig
}

type APIServer interface {
	Config() *Config

	// Add a WebService to the Container. It will detect duplicate root paths and exit in that case.
	Add(*restful.WebService) *restful.Container
	// Remove a WebService from the Container.
	Remove(service *restful.WebService) error
	// Handle registers the handler for the given pattern.
	// If a handler already exists for pattern, Handle panics.
	Handle(path string, handler http.Handler)
	// UnlistedHandle registers the handler for the given pattern, but doesn't list it.
	// If a handler already exists for pattern, Handle panics.
	UnlistedHandle(path string, handler http.Handler)
	// HandlePrefix is like Handle, but matches for anything under the path.  Like a standard golang trailing slash.
	HandlePrefix(path string, handler http.Handler)
	// UnlistedHandlePrefix is like UnlistedHandle, but matches for anything under the path.  Like a standard golang trailing slash.
	UnlistedHandlePrefix(path string, handler http.Handler)
	// ListedPaths is an alphabetically sorted list of paths to be reported at /.
	ListedPaths() []string

	// Filter appends a container FilterFunction. These are called before dispatching
	// a http.Request to a WebService from the container
	Filter(restful.FilterFunction)

	Serializer() runtime.NegotiatedSerializer
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
	Anonymous     bool
}

func (s *SecureServingInfo) ApplyClientCert(clientCA dynamiccertificates.CAContentProvider) error {
	if s == nil {
		return nil
	}
	if clientCA == nil {
		return nil
	}
	if s.ClientCA == nil {
		s.ClientCA = clientCA
		return nil
	}

	s.ClientCA = dynamiccertificates.NewUnionCAContentProvider(s.ClientCA, clientCA)
	return nil
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

type AuthorizationInfo struct {
	// Authorizer determines whether the subject is allowed to make the request based only
	// on the RequestURI
	Authorizer authorizer.Authorizer
	Modes      sets.String
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
		ParameterCodec:       scheme.ParameterCodec,
	}
}

func DefaultBuildHandlerChain(apiHandler http.Handler, c *Config) http.Handler {
	handler := apiHandler

	handler = filters.TrackCompleted(apiHandler)
	handler = filters.WithAuthorization(handler, c.Authorization.Authorizer, c.Serializer)
	handler = filters.TrackStarted(handler, c.TracerProvider, "authorization")

	//if s.FlowControl != nil {
	//	handler = filterlatency.TrackCompleted(handler)
	//	handler = genericfilters.WithPriorityAndFairness(handler, c.LongRunningFunc, c.FlowControl)
	//	handler = filterlatency.TrackStarted(handler, "priorityandfairness")
	//} else {
	//	handler = genericfilters.WithMaxInFlightLimit(handler, c.MaxRequestsInFlight, c.MaxMutatingRequestsInFlight, c.LongRunningFunc)
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
	handler = filters.WithAuthentication(handler, c.Authentication.Authenticator, failedHandler, c.Authentication.APIAudiences, c.RequestHeaderConfig)
	handler = filters.TrackStarted(handler, c.TracerProvider, "authentication")

	handler = sessions.WithSessions(handler)

	handler = filters.WithCORS(handler, c.CorsAllowedOriginList, nil, nil, nil, "true")

	// WithTimeoutForNonLongRunningRequests will call the rest of the request handling in a go-routine with the
	// context with deadline. The go-routine can keep running, while the timeout logic will return a timeout to the client.
	handler = filters.WithTimeoutForNonLongRunningRequests(handler, c.LongRunningFunc)

	handler = filters.WithRequestDeadline(handler, c.AuditBackend, c.AuditPolicyRuleEvaluator, c.LongRunningFunc, c.Serializer, c.RequestTimeout)
	handler = filters.WithWaitGroup(handler, c.LongRunningFunc, c.HandlerChainWaitGroup)
	//if s.SecureServing != nil && s.GoawayChance > 0 {
	//	handler = filters.WithProbabilisticGoaway(handler, s.GoawayChance)
	//}
	handler = filters.WithWarningRecorder(handler)
	handler = filters.WithCacheControl(handler)
	handler = filters.WithHSTS(handler, c.HSTSDirectives)

	handler = filters.WithHTTPLogging(handler)
	handler = tracing.WithTracing(handler, tracing.WithTracerProvider(c.TracerProvider))
	handler = filters.WithLatencyTrackers(handler)
	handler = filters.WithRequestInfo(handler, c.RequestInfoResolver)
	handler = filters.WithRequestReceivedTimestamp(handler)
	handler = filters.WithHttpDump(handler)
	handler = filters.WithPanicRecovery(handler, c.RequestInfoResolver)
	handler = filters.WithAuditInit(handler)
	return handler
}
