package config

import (
	"fmt"
	"net"
	"strings"

	"github.com/yubo/golib/api"
	"github.com/yubo/golib/configer"
	"github.com/yubo/golib/util/errors"
	utilerrors "github.com/yubo/golib/util/errors"
)

func NewServerRunOptions() *ServerRunOptions {
	return &ServerRunOptions{
		MaxRequestsInFlight:         400,
		MaxMutatingRequestsInFlight: 200,
		RequestTimeout:              api.NewDuration("60s"),
		LivezGracePeriod:            api.NewDuration("0s"),
		MinRequestTimeout:           api.NewDuration("1800s"),
		ShutdownDelayDuration:       api.NewDuration("0s"),
		JSONPatchMaxCopyBytes:       3 * 1024 * 1024,
		MaxRequestBodyBytes:         3 * 1024 * 1024,
		EnablePriorityAndFairness:   true,
	}
}

// ServerRunOptions contains the options while running a generic api server.
type ServerRunOptions struct {
	AdvertiseAddress net.IP `json:"advertiseAddress" flag:"advertise-address" description:"The IP address on which to advertise the apiserver to members of the cluster. This address must be reachable by the rest of the cluster. If blank, the --bind-address will be used. If --bind-address is unspecified, the host's default interface will be used."`

	CorsAllowedOriginList       []string     `json:"corsAllowedOriginList"`
	HSTSDirectives              []string     `json:"hstsDirectives" flag:"strict-transport-security-directives" description:"List of directives for HSTS, comma separated. If this list is empty, then HSTS directives will not be added. Example: 'max-age=31536000,includeSubDomains,preload'"`
	ExternalHost                string       `json:"externalHost" flag:"external-hostname" description:"The hostname to use when generating externalized URLs for this master (e.g. Swagger API Docs or OpenID Discovery)."`
	MaxRequestsInFlight         int          `json:"maxRequestsInFlight" default:"400" flag:"max-requests-inflight" description:"The maximum number of non-mutating requests in flight at a given time. When the server exceeds this, it rejects requests. Zero for no limit."`
	MaxMutatingRequestsInFlight int          `json:"maxMutatingRequestsInFlight" default:"200" flag:"max-mutating-requests-inflight" description:"The maximum number of mutating requests in flight at a given time. When the server exceeds this, it rejects requests. Zero for no limit."`
	RequestTimeout              api.Duration `json:"requestTimeout" default:"60s" flag:"request-timeout" description:"An optional field indicating the duration a handler must keep a request open before timing it out. This is the default request timeout for requests but may be overridden by flags such as --min-request-timeout for specific types of requests."`
	GoawayChance                float64      `json:"goawayChance" flag:"goaway-chance" description:"To prevent HTTP/2 clients from getting stuck on a single apiserver, randomly close a connection (GOAWAY). The client's other in-flight requests won't be affected, and the client will reconnect, likely landing on a different apiserver after going through the load balancer again. This argument sets the fraction of requests that will be sent a GOAWAY. Clusters with single apiservers, or which don't use a load balancer, should NOT enable this. Min is 0 (off), Max is .02 (1/50 requests); .001 (1/1000) is a recommended starting point."`
	LivezGracePeriod            api.Duration `json:"livezGracePeriod" flag:"livez-grace-period" description:"This option represents the maximum amount of time it should take for apiserver to complete its startup sequence and become live. From apiserver's start time to when this amount of time has elapsed, /livez will assume that unfinished post-start hooks will complete successfully and therefore return true."`
	MinRequestTimeout           api.Duration `json:"minRequestTimeout" default:"1800s" flag:"min-request-timeout" description:"An optional field indicating the minimum number of seconds a handler must keep a request open before timing it out. Currently only honored by the watch request handler, which picks a randomized value above this number as the connection timeout, to spread out load."`
	ShutdownDelayDuration       api.Duration `json:"shutdownDelayDuration" flag:"shutdown-delay-duration" description:"Time to delay the termination. During that time the server keeps serving requests normally. The endpoints /healthz and /livez will return success, but /readyz immediately returns failure. Graceful termination starts after this delay has elapsed. This can be used to allow load balancer to stop sending traffic to this server."`
	// We intentionally did not add a flag for this option. Users of the
	// apiserver library can wire it to a flag.

	JSONPatchMaxCopyBytes int64 `json:"-"`
	// The limit on the request body size that would be accepted and
	// decoded in a write request. 0 means no limit.
	// We intentionally did not add a flag for this option. Users of the
	// apiserver library can wire it to a flag.
	MaxRequestBodyBytes       int64 `json:"maxRequestBodyBytes" flag:"max-resource-write-bytes" description:"The limit on the request body size that would be accepted and decoded in a write request."`
	EnablePriorityAndFairness bool  `json:"enablePriorityAndFairness" default:"true" flag:"enable-priority-and-fairness" description:"If true and the APIPriorityAndFairness feature gate is enabled, replace the max-in-flight handler with an enhanced one that queues and dispatches with priority and fairness"`
}

func (p *ServerRunOptions) GetTags() map[string]*configer.FieldTag {
	return nil
}

func (c *ServerRunOptions) Validate() error {
	errors := []error{}

	if c.LivezGracePeriod.Duration < 0 {
		errors = append(errors, fmt.Errorf("--livez-grace-period can not be a negative value"))
	}

	if c.MaxRequestsInFlight < 0 {
		errors = append(errors, fmt.Errorf("--max-requests-inflight can not be negative value"))
	}
	if c.MaxMutatingRequestsInFlight < 0 {
		errors = append(errors, fmt.Errorf("--max-mutating-requests-inflight can not be negative value"))
	}

	if c.RequestTimeout.Duration < 0 {
		errors = append(errors, fmt.Errorf("--request-timeout can not be negative value"))
	}

	if c.GoawayChance < 0 || c.GoawayChance > 0.02 {
		errors = append(errors, fmt.Errorf("--goaway-chance can not be less than 0 or greater than 0.02"))
	}

	if c.MinRequestTimeout.Duration < 0 {
		errors = append(errors, fmt.Errorf("--min-request-timeout can not be negative value"))
	}

	if c.ShutdownDelayDuration.Duration < 0 {
		errors = append(errors, fmt.Errorf("--shutdown-delay-duration can not be negative value"))
	}

	if c.MaxRequestBodyBytes < 0 {
		errors = append(errors, fmt.Errorf("--max-resource-write-bytes can not be negative value"))
	}

	if err := validateHSTSDirectives(c.HSTSDirectives); err != nil {
		errors = append(errors, err)
	}

	return utilerrors.NewAggregate(errors)
}

func validateHSTSDirectives(hstsDirectives []string) error {
	// HSTS Headers format: Strict-Transport-Security:max-age=expireTime [;includeSubDomains] [;preload]
	// See https://tools.ietf.org/html/rfc6797#section-6.1 for more information
	allErrors := []error{}
	for _, hstsDirective := range hstsDirectives {
		if len(strings.TrimSpace(hstsDirective)) == 0 {
			allErrors = append(allErrors, fmt.Errorf("empty value in strict-transport-security-directives"))
			continue
		}
		if hstsDirective != "includeSubDomains" && hstsDirective != "preload" {
			maxAgeDirective := strings.Split(hstsDirective, "=")
			if len(maxAgeDirective) != 2 || maxAgeDirective[0] != "max-age" {
				allErrors = append(allErrors, fmt.Errorf("--strict-transport-security-directives invalid, allowed values: max-age=expireTime, includeSubDomains, preload. see https://tools.ietf.org/html/rfc6797#section-6.1 for more information"))
			}
		}
	}
	return errors.NewAggregate(allErrors)
}

// DefaultAdvertiseAddress sets the field AdvertiseAddress if unset. The field will be set based on the SecureServingOptions.
func (s *ServerRunOptions) DefaultAdvertiseAddress(secure *SecureServingOptions) error {
	if secure == nil {
		return nil
	}

	if s.AdvertiseAddress == nil || s.AdvertiseAddress.IsUnspecified() {
		hostIP, err := secure.DefaultExternalAddress()
		if err != nil {
			return fmt.Errorf("Unable to find suitable network address.error='%v'. "+
				"Try to set the AdvertiseAddress directly or provide a valid BindAddress to fix this.", err)
		}
		s.AdvertiseAddress = hostIP
	}

	return nil
}
