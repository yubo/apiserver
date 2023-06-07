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

package filters

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/yubo/apiserver/pkg/authentication/authenticator"
	"github.com/yubo/apiserver/pkg/authentication/authenticatorfactory"
	"github.com/yubo/apiserver/pkg/authentication/request/headerrequest"
	genericapirequest "github.com/yubo/apiserver/pkg/request"
	"github.com/yubo/apiserver/pkg/responsewriters"
	apierrors "github.com/yubo/golib/api/errors"
	"github.com/yubo/golib/runtime"
	"k8s.io/klog/v2"
)

type authenticationRecordMetricsFunc func(context.Context, *authenticator.Response, bool, error, authenticator.Audiences, time.Time, time.Time)

// WithAuthentication creates an http handler that tries to authenticate the given request as a user, and then
// stores any such user found onto the provided context for the request. If authentication fails or returns an error
// the failed handler is used. On success, "Authorization" header is removed from the request and handler
// is invoked to serve the request.
func WithAuthentication(handler http.Handler, auth authenticator.Request, failed http.Handler, apiAuds authenticator.Audiences, requestHeaderConfig *authenticatorfactory.RequestHeaderConfig) http.Handler {
	return withAuthentication(handler, auth, failed, apiAuds, requestHeaderConfig, recordAuthenticationMetrics)
}

func withAuthentication(handler http.Handler, auth authenticator.Request, failed http.Handler, apiAuds authenticator.Audiences, requestHeaderConfig *authenticatorfactory.RequestHeaderConfig, metrics authenticationRecordMetricsFunc) http.Handler {
	if auth == nil {
		klog.Warning("Authentication is disabled")
		return handler
	}
	standardRequestHeaderConfig := &authenticatorfactory.RequestHeaderConfig{
		UsernameHeaders:     headerrequest.StaticStringSlice{"X-Remote-User"},
		GroupHeaders:        headerrequest.StaticStringSlice{"X-Remote-Group"},
		ExtraHeaderPrefixes: headerrequest.StaticStringSlice{"X-Remote-Extra-"},
	}
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		authenticationStart := time.Now()

		if len(apiAuds) > 0 {
			req = req.WithContext(authenticator.WithAudiences(req.Context(), apiAuds))
		}
		resp, ok, err := auth.AuthenticateRequest(req)
		authenticationFinish := time.Now()
		defer func() {
			metrics(req.Context(), resp, ok, err, apiAuds, authenticationStart, authenticationFinish)
		}()
		if err != nil || !ok {
			if err != nil {
				klog.ErrorS(err, "Unable to authenticate the request")
			}
			failed.ServeHTTP(w, req)
			return
		}

		if !audiencesAreAcceptable(apiAuds, resp.Audiences) {
			err = fmt.Errorf("unable to match the audience: %v , accepted: %v", resp.Audiences, apiAuds)
			klog.Error(err)
			failed.ServeHTTP(w, req)
			return
		}

		// authorization header is not required anymore in case of a successful authentication.
		req.Header.Del("Authorization")

		// delete standard front proxy headers
		headerrequest.ClearAuthenticationHeaders(
			req.Header,
			standardRequestHeaderConfig.UsernameHeaders,
			standardRequestHeaderConfig.GroupHeaders,
			standardRequestHeaderConfig.ExtraHeaderPrefixes,
		)

		// also delete any custom front proxy headers
		if requestHeaderConfig != nil {
			headerrequest.ClearAuthenticationHeaders(
				req.Header,
				requestHeaderConfig.UsernameHeaders,
				requestHeaderConfig.GroupHeaders,
				requestHeaderConfig.ExtraHeaderPrefixes,
			)
		}

		req = req.WithContext(genericapirequest.WithUser(req.Context(), resp.User))
		handler.ServeHTTP(w, req)
	})
}

func Unauthorized(s runtime.NegotiatedSerializer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		responsewriters.ErrorNegotiated(apierrors.NewUnauthorized("Unauthorized"), s, w, req)
	})
}

func audiencesAreAcceptable(apiAuds, responseAudiences authenticator.Audiences) bool {
	if len(apiAuds) == 0 || len(responseAudiences) == 0 {
		return true
	}

	return len(apiAuds.Intersect(responseAudiences)) > 0
}

// WithAuthentication creates an http handler that tries to authenticate the given request as a user, and then
// stores any such user found onto the provided context for the request. If authentication fails or returns an error
// the failed handler is used. On success, "Authorization" header is removed from the request and handler
// is invoked to serve the request.
//func WithAuthentication(handler http.Handler, auth authenticator.Request, failed http.Handler, apiAuds authenticator.Audiences, keepAuthoriztionHeader bool) http.Handler {
//	if auth == nil {
//		klog.V(1).Info("Authentication is disabled")
//		return handler
//	}
//	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
//		klog.V(5).Infof("entering filters.WithAuthentication")
//		defer klog.V(5).Infof("leaving filters.WithAuthentication")
//		authenticationStart := time.Now()
//
//		if len(apiAuds) > 0 {
//			req = req.WithContext(authenticator.WithAudiences(req.Context(), apiAuds))
//		}
//		resp, ok, err := auth.AuthenticateRequest(req)
//		defer recordAuthMetrics(req.Context(), resp, ok, err /*apiAuds,*/, authenticationStart)
//		if err != nil || !ok {
//			if err != nil {
//				klog.ErrorS(err, "Unable to authenticate the request")
//			}
//			failed.ServeHTTP(w, req)
//			return
//		}
//
//		if !audiencesAreAcceptable(apiAuds, resp.Audiences) {
//			err = fmt.Errorf("unable to match the audience: %v , accepted: %v", resp.Audiences, apiAuds)
//			klog.Error(err)
//			failed.ServeHTTP(w, req)
//			return
//		}
//
//		// authorization header is not required anymore in case of a successful authentication.
//		if !keepAuthoriztionHeader {
//			req.Header.Del("Authorization")
//		}
//
//		req = req.WithContext(genericapirequest.WithUser(req.Context(), resp.User))
//		klog.V(8).InfoS("leaving authn filter", "user", resp.User.GetName(), "groups", resp.User.GetGroups())
//		handler.ServeHTTP(w, req)
//	})
//}
//
//func Unauthorized(s runtime.NegotiatedSerializer) http.Handler {
//	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
//
//		responsewriters.ErrorNegotiated(apierrors.NewUnauthorized("Unauthorized"), s, w, req)
//	})
//}
//
//func audiencesAreAcceptable(apiAuds, responseAudiences authenticator.Audiences) bool {
//	if len(apiAuds) == 0 || len(responseAudiences) == 0 {
//		return true
//	}
//
//	return len(apiAuds.Intersect(responseAudiences)) > 0
//}
