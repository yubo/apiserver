package server

import (
	"context"
	"fmt"
	"os"

	"github.com/yubo/apiserver/components/version"
	"github.com/yubo/apiserver/pkg/authorization/authorizer"
	"github.com/yubo/apiserver/pkg/filters"
	genericapiserver "github.com/yubo/apiserver/pkg/server"
	"github.com/yubo/apiserver/pkg/server/options"
	"github.com/yubo/golib/scheme"
	"github.com/yubo/golib/util"
	"github.com/yubo/golib/util/sets"
	"go.opentelemetry.io/otel"
	"k8s.io/klog/v2"
)

// completedServerRunOptions is a private wrapper that enforces a call of Complete() before Run can be invoked.
type completedServerRunOptions struct {
	*Config
}

// complete set default ServerRunOptions.
// Should be called after kube-apiserver flags parsed.
func complete(ctx context.Context, s *Config) (completedServerRunOptions, error) {
	var options completedServerRunOptions
	// set defaults
	if err := s.GenericServerRunOptions.DefaultAdvertiseAddress(s.SecureServing.SecureServingOptions); err != nil {
		return options, err
	}

	if err := s.SecureServing.MaybeDefaultWithSelfSignedCerts(
		s.GenericServerRunOptions.AdvertiseAddress.String(),
		s.SelfSignedCerts.AlternateDNS,
		s.SelfSignedCerts.AlternateIPs); err != nil {
		return options, fmt.Errorf("error creating self-signed certificates: %v", err)
	}

	if len(s.GenericServerRunOptions.ExternalHost) == 0 {
		if len(s.GenericServerRunOptions.AdvertiseAddress) > 0 {
			s.GenericServerRunOptions.ExternalHost = s.GenericServerRunOptions.AdvertiseAddress.String()
		} else {
			hostname, err := os.Hostname()
			if err != nil {
				return options, fmt.Errorf("error finding host name: %v", err)
			}
			s.GenericServerRunOptions.ExternalHost = hostname
		}
		klog.Infof("external host was not specified, using %v", s.GenericServerRunOptions.ExternalHost)
	}

	if s.SecureServing != nil && !util.BoolValue(s.SecureServing.Enabled) {
		s.SecureServing = nil
	}

	options.Config = s
	return options, nil
}

// buildGenericConfig takes the master server options and produces the genericapiserver.Config associated with it
func buildGenericConfig(ctx context.Context, s *Config) (genericConfig *genericapiserver.Config, lastErr error) {
	genericConfig = genericapiserver.NewConfig(scheme.Codecs)
	//genericConfig.MergedResourceConfig = controlplane.DefaultAPIResourceConfigSource()

	genericConfig.CorsAllowedOriginList = s.GenericServerRunOptions.CorsAllowedOriginList
	genericConfig.HSTSDirectives = s.GenericServerRunOptions.HSTSDirectives
	genericConfig.RequestTimeout = s.GenericServerRunOptions.RequestTimeout.Duration
	genericConfig.ShutdownDelayDuration = s.GenericServerRunOptions.ShutdownDelayDuration.Duration
	if genericConfig.SecuritySchemes, lastErr = options.ToSpecSecuritySchemes(s.GenericServerRunOptions.SecuritySchemes); lastErr != nil {
		return
	}

	if lastErr = s.GenericServerRunOptions.ApplyTo(genericConfig); lastErr != nil {
		return
	}

	if lastErr = s.SecureServing.ApplyTo(&genericConfig.SecureServing, &genericConfig.LoopbackClientConfig); lastErr != nil {
		return
	}
	if lastErr = s.InsecureServing.ApplyToWithLoopback2(&genericConfig.InsecureServing, &genericConfig.LoopbackClientConfig); lastErr != nil {
		return
	}
	if lastErr = s.Features.ApplyTo(genericConfig); lastErr != nil {
		return
	}

	genericConfig.LongRunningFunc = filters.BasicLongRunningRequestCheck(
		sets.NewString("watch", "proxy"),
		sets.NewString("attach", "exec", "proxy", "log", "portforward"),
	)
	kubeVersion := version.Get()
	genericConfig.Version = &kubeVersion

	// Use protobufs for self-communication.
	// Since not every generic apiserver has to support protobufs, we
	// cannot default to it in generic apiserver and need to explicitly
	// set it in kube-apiserver.
	genericConfig.LoopbackClientConfig.ContentConfig.ContentType = "application/vnd.kubernetes.protobuf"
	// Disable compression for self-communication, since we are going to be
	// on a fast local network
	genericConfig.LoopbackClientConfig.DisableCompression = true

	// Authentication.ApplyTo requires already applied OpenAPIConfig and EgressSelector if present
	if lastErr = s.Authentication.ApplyTo(ctx, &genericConfig.Authentication, genericConfig.SecureServing); lastErr != nil {
		return
	}

	var err error
	genericConfig.Authorization.Authorizer, genericConfig.RuleResolver, err = BuildAuthorizer(s)
	if err != nil {
		lastErr = fmt.Errorf("invalid authorization config: %v", err)
		return
	}

	lastErr = s.Audit.ApplyTo(genericConfig)
	if lastErr != nil {
		return
	}

	genericConfig.TracerProvider = otel.GetTracerProvider()

	return
}

// BuildAuthorizer constructs the authorizer
func BuildAuthorizer(s *Config) (authorizer.Authorizer, authorizer.RuleResolver, error) {
	authorizationConfig := s.Authorization.ToAuthorizationConfig()

	return authorizationConfig.New()
}

// deprecated
func authInit(c *genericapiserver.Config) error {
	//c.TracerProvider = otel.GetTracerProvider()

	//if audit, _ := dbus.GetAuditor(); audit != nil {
	//	c.AuditBackend = audit.Backend()
	//	c.AuditPolicyRuleEvaluator = audit.AuditPolicyRuleEvaluator()
	//}

	//if authz, _ := dbus.GetAuthorizationInfo(); authz != nil {
	//	c.Authorization = authz
	//} else {
	//	c.Authorization = &server.AuthorizationInfo{
	//		Authorizer: authorizerfactory.NewAlwaysAllowAuthorizer(),
	//		Modes:      sets.NewString("AlwaysAllow"),
	//	}
	//}
	//klog.V(3).InfoS("Authorizer", "modes", c.Authorization.Modes)

	//if authn, _ := dbus.GetAuthenticationInfo(); authn != nil {
	//	c.Authentication = authn
	//} else {
	//	c.Authentication = &server.AuthenticationInfo{}
	//}

	//if rhc, _ := dbus.GetRequestHeaderConfig(); rhc != nil {
	//	c.Authentication.RequestHeaderConfig = rhc
	//}

	// ApplyAuthorization will conditionally modify the authentication options based on the authorization options
	// authorization ModeAlwaysAllow cannot be combined with AnonymousAuth.
	// in such a case the AnonymousAuth is stomped to false and you get a message
	//if c.Authorization != nil && c.Authentication != nil &&
	//	c.Authentication.Anonymous &&
	//	c.Authorization.Modes.Has("AlwaysAllow") {
	//	return fmt.Errorf("AnonymousAuth is not allowed with the AlwaysAllow authorizer. Resetting AnonymousAuth to false. You should use a different authorizer")
	//}

	//genericapiserver.AuthorizeClientBearerToken(c.LoopbackClientConfig, c.Authentication, c.Authorization)

	return nil
}
