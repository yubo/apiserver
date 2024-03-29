/*
Copyright 2020 The Kubernetes Authors.

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
	"errors"
	"fmt"
	"net/http"
	"time"

	auditinternal "github.com/yubo/apiserver/pkg/apis/audit"
	"github.com/yubo/apiserver/pkg/audit"
	"github.com/yubo/apiserver/pkg/audit/policy"
	"github.com/yubo/apiserver/pkg/request"
	"github.com/yubo/apiserver/pkg/responsewriters"
	"github.com/yubo/golib/api"
	apierrors "github.com/yubo/golib/api/errors"
	"github.com/yubo/golib/runtime"
	utilclock "github.com/yubo/golib/util/clock"
	utilruntime "github.com/yubo/golib/util/runtime"
	"k8s.io/klog/v2"
)

const (
	// The 'timeout' query parameter in the request URL has an invalid duration specifier
	invalidTimeoutInURL = "invalid timeout specified in the request URL"
)

// WithRequestDeadline determines the timeout duration applicable to the given request and sets a new context
// with the appropriate deadline.
// auditWrapper provides an http.Handler that audits a failed request.
// longRunning returns true if he given request is a long running request.
// requestTimeoutMaximum specifies the default request timeout value.
func WithRequestDeadline(handler http.Handler, sink audit.Sink, policy policy.Checker,
	longRunning request.LongRunningRequestCheck,
	s runtime.NegotiatedSerializer,
	requestTimeoutMaximum time.Duration,
) http.Handler {
	return withRequestDeadline(handler, sink, policy, longRunning, s, requestTimeoutMaximum, utilclock.RealClock{})
}

func withRequestDeadline(handler http.Handler, sink audit.Sink, policy policy.Checker, longRunning request.LongRunningRequestCheck,
	s runtime.NegotiatedSerializer, requestTimeoutMaximum time.Duration, clock utilclock.PassiveClock) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()

		requestInfo, ok := request.RequestInfoFrom(ctx)
		if !ok {
			handleError(w, req, http.StatusInternalServerError, fmt.Errorf("no RequestInfo found in context, handler chain must be wrong"))
			return
		}
		if longRunning(req, requestInfo) {
			handler.ServeHTTP(w, req)
			return
		}

		userSpecifiedTimeout, ok, err := parseTimeout(req)
		if err != nil {
			statusErr := apierrors.NewBadRequest(fmt.Sprintf("%s", err.Error()))

			klog.Errorf("Error - %s: %#v", err.Error(), req.RequestURI)

			failed := failedErrorHandler(s, statusErr)
			failWithAudit := withFailedRequestAudit(failed, statusErr, sink, policy)
			failWithAudit.ServeHTTP(w, req)
			return
		}

		timeout := requestTimeoutMaximum
		if ok {
			// we use the default timeout enforced by the apiserver:
			// - if the user has specified a timeout of 0s, this implies no timeout on the user's part.
			// - if the user has specified a timeout that exceeds the maximum deadline allowed by the apiserver.
			if userSpecifiedTimeout > 0 && userSpecifiedTimeout < requestTimeoutMaximum {
				timeout = userSpecifiedTimeout
			}
		}

		started := clock.Now()
		if requestStartedTimestamp, ok := request.ReceivedTimestampFrom(ctx); ok {
			started = requestStartedTimestamp
		}

		ctx, cancel := context.WithDeadline(ctx, started.Add(timeout))
		defer cancel()

		req = req.WithContext(ctx)
		handler.ServeHTTP(w, req)
	})
}

// withFailedRequestAudit decorates a failed http.Handler and is used to audit a failed request.
// statusErr is used to populate the Message property of ResponseStatus.
func withFailedRequestAudit(failedHandler http.Handler, statusErr *apierrors.StatusError, sink audit.Sink, policy policy.Checker) http.Handler {
	if sink == nil || policy == nil {
		return failedHandler
	}
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		req, ev, omitStages, err := createAuditEventAndAttachToContext(req, policy)
		if err != nil {
			utilruntime.HandleError(fmt.Errorf("failed to create audit event: %v", err))
			responsewriters.InternalError(w, req, errors.New("failed to create audit event"))
			return
		}
		if ev == nil {
			failedHandler.ServeHTTP(w, req)
			return
		}

		ev.ResponseStatus = &api.Status{}
		ev.Stage = auditinternal.StageResponseStarted
		if statusErr != nil {
			ev.ResponseStatus.Message = statusErr.Error()
		}

		rw := decorateResponseWriter(req.Context(), w, ev, sink, omitStages)
		failedHandler.ServeHTTP(rw, req)
	})
}

// failedErrorHandler returns an http.Handler that uses the specified StatusError object
// to render an error response to the request.
func failedErrorHandler(s runtime.NegotiatedSerializer, statusErr *apierrors.StatusError) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		responsewriters.ErrorNegotiated(statusErr, s, w, req)
		//ctx := req.Context()
		//requestInfo, found := request.RequestInfoFrom(ctx)
		//if !found {
		//	responsewriters.InternalError(w, req, errors.New("no RequestInfo found in the context"))
		//	return
		//}

		//gv := schema.GroupVersion{Group: requestInfo.APIGroup, Version: requestInfo.APIVersion}
		//responsewriters.ErrorNegotiated(statusError, s, gv, w, req)
	})
}

// parseTimeout parses the given HTTP request URL and extracts the timeout query parameter
// value if specified by the user.
// If a timeout is not specified the function returns false and err is set to nil
// If the value specified is malformed then the function returns false and err is set
func parseTimeout(req *http.Request) (time.Duration, bool, error) {
	value := req.URL.Query().Get("timeout")
	if value == "" {
		return 0, false, nil
	}

	timeout, err := time.ParseDuration(value)
	if err != nil {
		return 0, false, fmt.Errorf("%s - %s", invalidTimeoutInURL, err.Error())
	}

	return timeout, true, nil
}

func handleError(w http.ResponseWriter, r *http.Request, code int, err error) {
	errorMsg := fmt.Sprintf("Error - %s: %#v", err.Error(), r.RequestURI)
	http.Error(w, errorMsg, code)
	klog.Errorf(errorMsg)
}
