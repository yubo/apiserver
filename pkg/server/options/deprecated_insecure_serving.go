/*
Copyright 2017 The Kubernetes Authors.

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

package options

import (
	"fmt"
	"net"

	"github.com/yubo/apiserver/pkg/server"
	"github.com/yubo/client-go/rest"
)

// DeprecatedInsecureServingOptions are for creating an unauthenticated, unauthorized, insecure port.
// No one should be using these anymore.
// DEPRECATED: all insecure serving options are removed in a future version
type DeprecatedInsecureServingOptions struct {
	Enabled     *bool  `json:"enabled" flag:"insecure-serving" default:"false" description:"enable the insecure serving"`
	BindAddress net.IP `json:"bindAddress" flag:"address" description:"The IP address on which to serve the --port (set to 0.0.0.0 or :: for listening in all interfaces and IP families)." deprecated:"This flag will be removed in a future version."`
	BindPort    int    `json:"bindPort" flag:"port" description:"The port on which to serve unsecured, unauthenticated access." deprecated:"This flag will be removed in a future version."`
	// BindNetwork is the type of network to bind to - defaults to "tcp", accepts "tcp",
	// "tcp4", and "tcp6".
	BindNetwork string `json:"bindNework" default:"tcp" description:"BindNetwork is the type of network to bind to - accepts \"tcp\", \"tcp4\", and \"tcp6\"."`

	// Listener is the secure server network listener.
	// either Listener or BindAddress/BindPort/BindNetwork is set,
	// if Listener is set, use it and omit BindAddress/BindPort/BindNetwork.
	Listener net.Listener `json:"-"`

	// ListenFunc can be overridden to create a custom listener, e.g. for mocking in tests.
	// It defaults to options.CreateListener.
	ListenFunc func(network, addr string, config net.ListenConfig) (net.Listener, int, error) `json:"-"`
}

func NewDeprecatedInsecureServingOptions() *DeprecatedInsecureServingOptions {
	return &DeprecatedInsecureServingOptions{
		BindAddress: net.ParseIP("0.0.0.0"),
		BindPort:    8080,
		BindNetwork: "tcp",
	}
}

// Validate ensures that the insecure port values within the range of the port.
func (p *DeprecatedInsecureServingOptions) Validate() []error {
	if p == nil {
		return nil
	}

	errors := []error{}

	if p.BindPort < 0 || p.BindPort > 65535 {
		errors = append(errors, fmt.Errorf("insecure port %v must be between 0 and 65535, inclusive. 0 for turning off insecure (HTTP) port", p.BindPort))
	}

	return errors
}

// ApplyTo adds DeprecatedInsecureServingOptions to the insecureserverinfo and kube-controller manager configuration.
// Note: the double pointer allows to set the *DeprecatedInsecureServingInfo to nil without referencing the struct hosting this pointer.
func (p *DeprecatedInsecureServingOptions) ApplyTo(c **server.DeprecatedInsecureServingInfo) error {
	if p == nil {
		return nil
	}
	if p.BindPort <= 0 {
		return nil
	}

	if p.Listener == nil {
		var err error
		listen := CreateListener
		if p.ListenFunc != nil {
			listen = p.ListenFunc
		}
		addr := net.JoinHostPort(p.BindAddress.String(), fmt.Sprintf("%d", p.BindPort))
		p.Listener, p.BindPort, err = listen(p.BindNetwork, addr, net.ListenConfig{})
		if err != nil {
			return fmt.Errorf("failed to create listener: %v", err)
		}
	}

	*c = &server.DeprecatedInsecureServingInfo{
		Listener: p.Listener,
	}

	return nil
}

// ApplyTo fills up serving information in the server configuration.
func (p *DeprecatedInsecureServingOptions) ApplyToWithLoopback(insecureServingInfo **server.DeprecatedInsecureServingInfo, loopbackClientConfig **rest.Config) error {
	if p == nil || insecureServingInfo == nil {
		return nil
	}

	if err := p.ApplyTo(insecureServingInfo); err != nil {
		return err
	}

	if *insecureServingInfo == nil || loopbackClientConfig == nil {
		return nil
	}

	secureLoopbackClientConfig, err := (*insecureServingInfo).NewLoopbackClientConfig()
	switch {
	// if we failed and there's no fallback loopback client config, we need to fail
	case err != nil && *loopbackClientConfig == nil:
		return err

		// if we failed, but we already have a fallback loopback client config (usually insecure), allow it
	case err != nil && *loopbackClientConfig != nil:

	default:
		*loopbackClientConfig = secureLoopbackClientConfig
	}

	return nil
}

// ApplyTo fills up serving information in the server configuration, skip apply to client config if already set.
func (p *DeprecatedInsecureServingOptions) ApplyToWithLoopback2(insecureServingInfo **server.DeprecatedInsecureServingInfo, loopbackClientConfig **rest.Config) error {
	if loopbackClientConfig == nil || *loopbackClientConfig != nil {
		return p.ApplyTo(insecureServingInfo)
	}

	return p.ApplyToWithLoopback(insecureServingInfo, loopbackClientConfig)
}
