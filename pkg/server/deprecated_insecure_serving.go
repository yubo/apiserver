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

package server

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/yubo/apiserver/pkg/authentication/authenticator"
	"github.com/yubo/apiserver/pkg/authentication/user"
	"github.com/yubo/client-go/rest"
	utilnet "github.com/yubo/golib/util/net"
	"k8s.io/klog/v2"
)

// DeprecatedInsecureServingInfo is the main context object for the insecure http server.
// HTTP does NOT include authentication or authorization.
// You shouldn't be using this.  It makes sig-auth sad.
type DeprecatedInsecureServingInfo struct {
	// Listener is the secure server network listener.
	Listener net.Listener
	// optional server name for log messages
	Name string
}

// Serve starts an insecure http server with the given handler. It fails only if
// the initial listen call fails. It does not block.
func (s *DeprecatedInsecureServingInfo) Serve(handler http.Handler, shutdownTimeout time.Duration, stopCh <-chan struct{}) (<-chan struct{}, error) {
	insecureServer := &http.Server{
		Addr:           s.Listener.Addr().String(),
		Handler:        handler,
		MaxHeaderBytes: 1 << 20,
	}

	if len(s.Name) > 0 {
		klog.Infof("Serving %s insecurely on %s", s.Name, s.Listener.Addr())
	} else {
		klog.Infof("Serving insecurely on %s", s.Listener.Addr())
	}
	return RunServer(insecureServer, s.Listener, shutdownTimeout, stopCh)
}

func (s *DeprecatedInsecureServingInfo) HostPort() (string, int, error) {
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

func (s *DeprecatedInsecureServingInfo) NewLoopbackClientConfig() (*rest.Config, error) {
	if s == nil {
		return nil, nil
	}

	host, port, err := LoopbackHostPort(s.Listener.Addr().String())
	if err != nil {
		return nil, err
	}

	return &rest.Config{
		Host: "http://" + net.JoinHostPort(host, port),
		// Increase QPS limits. The client is currently passed to all admission plugins,
		// and those can be throttled in case of higher load on apiserver - see #22340 and #22422
		// for more details. Once #22422 is fixed, we may want to remove it.
		QPS:   50,
		Burst: 100,
	}, nil
}

// InsecureSuperuser implements authenticator.Request to always return a superuser.
// This is functionally equivalent to skipping authentication and authorization,
// but allows apiserver code to stop special-casing a nil user to skip authorization checks.
type InsecureSuperuser struct{}

func (InsecureSuperuser) AuthenticateRequest(req *http.Request) (*authenticator.Response, bool, error) {
	auds, _ := authenticator.AudiencesFrom(req.Context())
	return &authenticator.Response{
		User: &user.DefaultInfo{
			Name:   "system:unsecured",
			Groups: []string{user.SystemPrivilegedGroup, user.AllAuthenticated},
		},
		Audiences: auds,
	}, true, nil
}
