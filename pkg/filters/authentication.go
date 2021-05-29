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
	"net/http"
	"time"

	"github.com/yubo/apiserver/pkg/authentication/authenticator"
	genericapirequest "github.com/yubo/apiserver/pkg/request"
	"github.com/yubo/apiserver/pkg/responsewriters"
	apierrors "github.com/yubo/golib/staging/api/errors"
	"k8s.io/klog/v2"
)

// WithAuthentication creates an http handler that tries to authenticate the given request as a user, and then
// stores any such user found onto the provided context for the request. If authentication fails or returns an error
// the failed handler is used. On success, "Authorization" header is removed from the request and handler
// is invoked to serve the request.
func WithAuthentication(handler http.Handler, auth authenticator.Request, failed http.Handler /*, apiAuds authenticator.Audiences*/) http.Handler {
	if auth == nil {
		klog.Warning("Authentication is disabled")
		return handler
	}
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		klog.V(8).Infof("entering authn filter")
		authenticationStart := time.Now()

		//if len(apiAuds) > 0 {
		//	req = req.WithContext(authenticator.WithAudiences(req.Context(), apiAuds))
		//}
		resp, ok, err := auth.AuthenticateRequest(req)
		klog.V(8).Infof("authn resp %+v ok %v err %v", resp, ok, err)
		defer recordAuthMetrics(req.Context(), resp, ok, err /*apiAuds,*/, authenticationStart)
		if err != nil || !ok {
			if err != nil {
				klog.ErrorS(err, "Unable to authenticate the request")
			}
			failed.ServeHTTP(w, req)
			return
		}

		//if !audiencesAreAcceptable(apiAuds, resp.Audiences) {
		//	err = fmt.Errorf("unable to match the audience: %v , accepted: %v", resp.Audiences, apiAuds)
		//	klog.Error(err)
		//	failed.ServeHTTP(w, req)
		//	return
		//}

		// authorization header is not required anymore in case of a successful authentication.
		req.Header.Del("Authorization")

		req = req.WithContext(genericapirequest.WithUser(req.Context(), resp.User))
		klog.V(8).Infof("leaving authn filter")
		handler.ServeHTTP(w, req)
	})
}

func Unauthorized() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		responsewriters.Error(apierrors.NewUnauthorized("Unauthorized"), w, req)
	})
}

func audiencesAreAcceptable(apiAuds, responseAudiences authenticator.Audiences) bool {
	if len(apiAuds) == 0 || len(responseAudiences) == 0 {
		return true
	}

	return len(apiAuds.Intersect(responseAudiences)) > 0
}
