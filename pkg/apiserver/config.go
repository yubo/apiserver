/*
Copyright 2014 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package options contains flags and options for initializing an apiserver
package apiserver

import (
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	apirequest "github.com/yubo/apiserver/pkg/request"
	"github.com/yubo/golib/util"
	"github.com/yubo/golib/util/errors"
	utilerrors "github.com/yubo/golib/util/errors"
	"github.com/yubo/golib/util/sets"
	"k8s.io/klog/v2"
)

func newConfig() *config {
	return &config{}
}

// config contains the config while running a generic api server.
type config struct {
	Enabled bool `json:"enabled" default:"true" flag:"apiserver-enable" description:"api server enable"`

	ExternalHost string `json:"externalHost" flag:"external-hostname" description:"The hostname to use when generating externalized URLs for this master (e.g. Swagger API Docs or OpenID Discovery)."`

	// ExternalAddress is the address advertised, even if BindAddress is a loopback. By default this
	// is set to BindAddress if the later no loopback, or to the first host interface address.
	ExternalAddress net.IP `json:"-"`

	Host string `json:"host" default:"0.0.0.0" flag:"bind-host" description:"The IP address on which to listen for the --bind-port port. The associated interface(s) must be reachable by the rest of the cluster, and by CLI/web clients. If blank or an unspecified address (0.0.0.0 or ::), all interfaces will be used."` // BindAddress

	Port int `json:"port" default:"8080" flag:"bind-port" description:"The port on which to serve HTTPS with authentication and authorization. It cannot be switched off with 0."` // BindPort is ignored when Listener is set, will serve https even with 0.

	Network string `json:"bindNetwork" flag:"cors-allowed-origins" description:"List of allowed origins for CORS, comma separated.  An allowed origin can be a regular expression to support subdomain matching. If this list is empty CORS will not be enabled."` // BindNetwork is the type of network to bind to - defaults to "tcp", accepts "tcp", "tcp4", and "tcp6".

	CorsAllowedOriginList []string `json:"corsAllowedOriginList"`

	HSTSDirectives []string `json:"hstsDirectives" flag:"strict-transport-security-directives" description:"List of directives for HSTS, comma separated. If this list is empty, then HSTS directives will not be added. Example: 'max-age=31536000,includeSubDomains,preload'"`

	MaxRequestsInFlight int `json:"maxRequestsInFlight" default:"400" flag:"max-requests-inflight" description:"The maximum number of non-mutating requests in flight at a given time. When the server exceeds this, it rejects requests. Zero for no limit."`

	MaxMutatingRequestsInFlight int `json:"maxMutatingRequestsInFlight" default:"200" flag:"max-mutating-requests-inflight" description:"The maximum number of mutating requests in flight at a given time. When the server exceeds this, it rejects requests. Zero for no limit."`

	RequestTimeout int `json:"requestTimeout" default:"60" flag:"request-timeout" description:"An optional field indicating the duration a handler must keep a request open before timing it out. This is the default request timeout for requests but may be overridden by flags such as --min-request-timeout for specific types of requests."`

	GoawayChance float64 `json:"goawayChance" flag:"goaway-chance" description:"To prevent HTTP/2 clients from getting stuck on a single apiserver, randomly close a connection (GOAWAY). The client's other in-flight requests won't be affected, and the client will reconnect, likely landing on a different apiserver after going through the load balancer again. This argument sets the fraction of requests that will be sent a GOAWAY. Clusters with single apiservers, or which don't use a load balancer, should NOT enable this. Min is 0 (off), Max is .02 (1/50 requests); .001 (1/1000) is a recommended starting point."`

	LivezGracePeriod  int `json:"livezGracePeriod" flag:"livez-grace-period" description:"This option represents the maximum amount of time it should take for apiserver to complete its startup sequence and become live. From apiserver's start time to when this amount of time has elapsed, /livez will assume that unfinished post-start hooks will complete successfully and therefore return true."`
	MinRequestTimeout int `json:"minRequestTimeout" default:"1800" flag:"min-request-timeout" description:"An optional field indicating the minimum number of seconds a handler must keep a request open before timing it out. Currently only honored by the watch request handler, which picks a randomized value above this number as the connection timeout, to spread out load."`

	ShutdownTimeout       int `json:"shutdownTimeout" default:"60" description:"ShutdownTimeout is the timeout used for server shutdown. This specifies the timeout before server gracefully shutdown returns."`
	ShutdownDelayDuration int `json:"shutdownDelayDuration" flag:"shutdown-delay-duration" description:"Time to delay the termination. During that time the server keeps serving requests normally. The endpoints /healthz and /livez will return success, but /readyz immediately returns failure. Graceful termination starts after this delay has elapsed. This can be used to allow load balancer to stop sending traffic to this server."`

	// The limit on the request body size that would be accepted and
	// decoded in a write request. 0 means no limit.
	// We intentionally did not add a flag for this option. Users of the
	// apiserver library can wire it to a flag.
	MaxRequestBodyBytes int64 `json:"maxRequestBodyBytes" default:"3145728" flag:"max-resource-write-bytes" description:"The limit on the request body size that would be accepted and decoded in a write request."`

	EnablePriorityAndFairness bool `json:"enablePriorityAndFairness" default:"true" flag:"enable-priority-and-fairness" description:"If true and the APIPriorityAndFairness feature gate is enabled, replace the max-in-flight handler with an enhanced one that queues and dispatches with priority and fairness"`

	// ExternalAddress is the host name to use for external (public internet) facing URLs (e.g. Swagger)
	// Will default to a value based on secure serving info and available ipv4 IPs.
	//ExternalAddress net.IP `json:"-"`

	// Listener is the secure server network listener.
	// either Listener or BindAddress/BindPort/BindNetwork is set,
	// if Listener is set, use it and omit BindAddress/BindPort/BindNetwork.
	Listener net.Listener `json:"-"`

	requestTimeout        time.Duration
	livezGracePeriod      time.Duration
	minRequestTimeout     time.Duration
	shutdownTimeout       time.Duration
	shutdownDelayDuration time.Duration

	// EgressSelector provides a lookup mechanism for dialing outbound connections.
	// It does so based on a EgressSelectorConfiguration which was read at startup.
	// EgressSelector *egressselector.EgressSelector

	//CorsAllowedOriginList []string
	//HSTSDirectives        []string
	// FlowControl, if not nil, gives priority and fairness to request handling
	// FlowControl utilflowcontrol.Interface

	EnableIndex     bool `json:"enableIndex" default:"true"`
	EnableProfiling bool `json:"enableProfiling" default:"false"`
	// EnableDiscovery bool
	// Requires generic profiling enabled
	EnableContentionProfiling bool `json:"enableContentionProfiling" default:"true"`
	EnableMetrics             bool `json:"enableMetrics" default:"true"`

	// audit
}

func (s *config) String() string {
	return util.Prettify(s)
}

// Validate will be called by config reader
func (c *config) Validate() error {
	if len(c.ExternalHost) == 0 {
		if hostname, err := os.Hostname(); err == nil {
			c.ExternalHost = hostname
		} else {
			return fmt.Errorf("error finding host name: %v", err)
		}
		klog.V(1).Infof("external host was not specified, using %v", c.ExternalHost)
	}

	errors := []error{}

	if c.LivezGracePeriod < 0 {
		errors = append(errors, fmt.Errorf("--livez-grace-period can not be a negative value"))
	}

	if c.MaxRequestsInFlight < 0 {
		errors = append(errors, fmt.Errorf("--max-requests-inflight can not be negative value"))
	}
	if c.MaxMutatingRequestsInFlight < 0 {
		errors = append(errors, fmt.Errorf("--max-mutating-requests-inflight can not be negative value"))
	}

	if c.RequestTimeout < 0 {
		errors = append(errors, fmt.Errorf("--request-timeout can not be negative value"))
	}

	if c.GoawayChance < 0 || c.GoawayChance > 0.02 {
		errors = append(errors, fmt.Errorf("--goaway-chance can not be less than 0 or greater than 0.02"))
	}

	if c.MinRequestTimeout < 0 {
		errors = append(errors, fmt.Errorf("--min-request-timeout can not be negative value"))
	}

	if c.ShutdownDelayDuration < 0 {
		errors = append(errors, fmt.Errorf("--shutdown-delay-duration can not be negative value"))
	}

	if c.MaxRequestBodyBytes < 0 {
		errors = append(errors, fmt.Errorf("--max-resource-write-bytes can not be negative value"))
	}

	if err := validateHSTSDirectives(c.HSTSDirectives); err != nil {
		errors = append(errors, err)
	}

	if c.Port < 1 || c.Port > 65535 {
		errors = append(errors, fmt.Errorf("--bind-port %v must be between 1 and 65535, inclusive. It cannot be turned off with 0", c.Port))
	}

	c.requestTimeout = duration(c.RequestTimeout)
	c.livezGracePeriod = duration(c.LivezGracePeriod)
	c.minRequestTimeout = duration(c.MinRequestTimeout)
	c.shutdownTimeout = duration(c.ShutdownTimeout)
	c.shutdownDelayDuration = duration(c.ShutdownDelayDuration)

	c.ExternalAddress = net.ParseIP(c.Host)

	return utilerrors.NewAggregate(errors)
}

func NewRequestInfoResolver(c *config) *apirequest.RequestInfoFactory {
	return &apirequest.RequestInfoFactory{
		APIPrefixes:          sets.NewString("api", "apis"),
		GrouplessAPIPrefixes: sets.NewString("api"),
	}
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

func duration(second int) time.Duration {
	return time.Duration(second) * time.Second
}
