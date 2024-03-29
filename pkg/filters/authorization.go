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

package filters

import (
	"context"
	"errors"
	"net/http"

	"k8s.io/klog/v2"

	"github.com/emicklei/go-restful/v3"
	"github.com/yubo/apiserver/pkg/audit"
	"github.com/yubo/apiserver/pkg/authorization/authorizer"
	"github.com/yubo/apiserver/pkg/request"
	"github.com/yubo/apiserver/pkg/responsewriters"
	"github.com/yubo/golib/runtime"
)

const (
	// Annotation key names set in advanced audit
	decisionAnnotationKey = "authorization.k8s.io/decision"
	reasonAnnotationKey   = "authorization.k8s.io/reason"

	// Annotation values set in advanced audit
	decisionAllow  = "allow"
	decisionForbid = "forbid"
	reasonError    = "internal error"
)

// WithAuthorizationCheck passes all authorized requests on to handler, and returns a forbidden error otherwise.
func WithAuthorization(handler http.Handler, a authorizer.Authorizer, s runtime.NegotiatedSerializer) http.Handler {
	if a == nil {
		klog.Warning("Authorization is disabled")
		return handler
	}
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		klog.V(10).Infof("entering filters.WithAuthorization")
		defer klog.V(10).Infof("leaving filters.WithAuthorization")
		ctx := req.Context()
		ae := request.AuditEventFrom(ctx)

		attributes, err := GetAuthorizerAttributes(ctx)
		if err != nil {
			responsewriters.InternalError(w, req, err)
			return
		}
		authorized, reason, err := a.Authorize(ctx, attributes)
		// an authorizer like RBAC could encounter evaluation errors and still allow the request, so authorizer decision is checked before error here.
		if authorized == authorizer.DecisionAllow {
			audit.LogAnnotation(ae, decisionAnnotationKey, decisionAllow)
			audit.LogAnnotation(ae, reasonAnnotationKey, reason)
			handler.ServeHTTP(w, req)
			return
		}
		if err != nil {
			audit.LogAnnotation(ae, reasonAnnotationKey, reasonError)
			responsewriters.InternalError(w, req, err)
			return
		}

		klog.V(4).InfoS("Forbidden", "URI", req.RequestURI, "Reason", reason)
		audit.LogAnnotation(ae, decisionAnnotationKey, decisionForbid)
		audit.LogAnnotation(ae, reasonAnnotationKey, reason)
		responsewriters.Forbidden(ctx, attributes, w, req, reason, s)
	})
}

func Authz(a authorizer.Authorizer, s runtime.NegotiatedSerializer) restful.FilterFunction {
	if a == nil {
		return nil
	}
	return func(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
		klog.V(10).Infof("entering filters.Authz")
		defer klog.V(10).Infof("leaving filters.Authz")

		ctx := req.Request.Context()
		ae := request.AuditEventFrom(ctx)

		attributes, err := GetAuthorizerAttributes(ctx)
		if err != nil {
			responsewriters.InternalError(resp, req.Request, err)
			return
		}
		authorized, reason, err := a.Authorize(ctx, attributes)
		// an authorizer like RBAC could encounter evaluation errors and still allow the request, so authorizer decision is checked before error here.
		if authorized == authorizer.DecisionAllow {
			audit.LogAnnotation(ae, decisionAnnotationKey, decisionAllow)
			audit.LogAnnotation(ae, reasonAnnotationKey, reason)
			//handler.ServeHTTP(w, req)
			chain.ProcessFilter(req, resp)

			return
		}
		if err != nil {
			audit.LogAnnotation(ae, reasonAnnotationKey, reasonError)
			responsewriters.InternalError(resp, req.Request, err)
			return
		}

		klog.V(4).InfoS("Forbidden", "URI", req.Request.RequestURI, "Reason", reason)
		audit.LogAnnotation(ae, decisionAnnotationKey, decisionForbid)
		audit.LogAnnotation(ae, reasonAnnotationKey, reason)
		responsewriters.Forbidden(ctx, attributes, resp, req.Request, reason, s)
	}
}

func GetAuthorizerAttributes(ctx context.Context) (authorizer.Attributes, error) {
	attribs := authorizer.AttributesRecord{}

	user, ok := request.UserFrom(ctx)
	if ok {
		attribs.User = user
	}

	requestInfo, found := request.RequestInfoFrom(ctx)
	if !found {
		return nil, errors.New("no RequestInfo found in the context")
	}

	// Start with common attributes that apply to resource and non-resource requests
	attribs.ResourceRequest = requestInfo.IsResourceRequest
	attribs.Path = requestInfo.Path
	attribs.Verb = requestInfo.Verb

	//attribs.APIGroup = requestInfo.APIGroup
	//attribs.APIVersion = requestInfo.APIVersion
	attribs.Resource = requestInfo.Resource
	attribs.Subresource = requestInfo.Subresource
	attribs.Namespace = requestInfo.Namespace
	attribs.Name = requestInfo.Name

	return &attribs, nil
}
