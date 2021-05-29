/*
Copyright 2016 The Kubernetes Authors.

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

// from k8s.io/apiserver/pkg/server/options/authentication.go
package module

import (
	"time"

	"github.com/spf13/pflag"
	"github.com/yubo/golib/staging/util/wait"
	"github.com/yubo/golib/util"
)

// config contains all authentication options for API Server
type config struct {
	//APIAudiences         []string      `yaml:"apiAudiences"`
	TokenSuccessCacheTTL time.Duration `yaml:"tokenSuccessCacheTTL"`
	TokenFailureCacheTTL time.Duration `yaml:"tokenFailureCacheTTL"`
	Anonymous            bool          `yaml:"anonymous"`
}

// TokenFileAuthenticationOptions contains token file authentication options for API Server
type TokenFileAuthenticationOptions struct {
	TokenFile string
}

// WebHookAuthenticationOptions contains web hook authentication options for API Server
type WebHookAuthenticationOptions struct {
	ConfigFile string
	Version    string
	CacheTTL   time.Duration

	// RetryBackoff specifies the backoff parameters for the authentication webhook retry logic.
	// This allows us to configure the sleep time at each iteration and the maximum number of retries allowed
	// before we fail the webhook call in order to limit the fan out that ensues when the system is degraded.
	RetryBackoff *wait.Backoff
}

// newConfig create a new BuiltInAuthenticationOptions, just set default token cache TTL
func newConfig() *config {
	return &config{
		TokenSuccessCacheTTL: 10 * time.Second,
		TokenFailureCacheTTL: 0 * time.Second,
		Anonymous:            true,
	}
}

func defaultConfig() *config {
	return newConfig()
}

func (o *config) changed() interface{} {
	if o == nil {
		return nil
	}
	return util.Diff2Map(defaultConfig(), o)
}

// Validate checks invalid config combination
func (o *config) Validate() error {
	return nil
}

// addFlags returns flags of authentication for a API Server
func (o *config) addFlags(fs *pflag.FlagSet) {
	fs.DurationVar(&o.TokenSuccessCacheTTL, "token-success-cache-ttl", o.TokenSuccessCacheTTL, "The duration to cache success token.")
	fs.DurationVar(&o.TokenFailureCacheTTL, "token-failure-cache-ttl", o.TokenFailureCacheTTL, "The duration to cache failure token.")

	fs.BoolVar(&o.Anonymous, "anonymous-auth", o.Anonymous, ""+
		"Enables anonymous requests to the secure port of the API server. "+
		"Requests that are not rejected by another authentication method are treated as anonymous requests. "+
		"Anonymous requests have a username of system:anonymous, and a group name of system:unauthenticated.")
}
