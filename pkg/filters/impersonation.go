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
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/yubo/apiserver/pkg/audit"
	"github.com/yubo/apiserver/pkg/authentication/serviceaccount"
	"github.com/yubo/apiserver/pkg/authentication/user"
	"github.com/yubo/apiserver/pkg/authorization/authorizer"
	"github.com/yubo/apiserver/pkg/request"
	"github.com/yubo/apiserver/pkg/responsewriters"
	"github.com/yubo/apiserver/pkg/server/httplog"
	"github.com/yubo/golib/api"
	"github.com/yubo/golib/runtime"
	"k8s.io/klog/v2"
)

// WithImpersonation is a filter that will inspect and check requests that attempt to change the user.Info for their requests
func WithImpersonation(handler http.Handler, a authorizer.Authorizer, s runtime.NegotiatedSerializer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		impersonationRequests, err := buildImpersonationRequests(req.Header)
		if err != nil {
			klog.V(4).Infof("%v", err)
			responsewriters.InternalError(w, req, err)
			return
		}
		if len(impersonationRequests) == 0 {
			handler.ServeHTTP(w, req)
			return
		}

		ctx := req.Context()
		requestor, exists := request.UserFrom(ctx)
		if !exists {
			responsewriters.InternalError(w, req, errors.New("no user found for request"))
			return
		}

		// if groups are not specified, then we need to look them up differently depending on the type of user
		// if they are specified, then they are the authority (including the inclusion of system:authenticated/system:unauthenticated groups)
		groupsSpecified := len(req.Header[api.ImpersonateGroupHeader]) > 0

		// make sure we're allowed to impersonate each thing we're requesting.  While we're iterating through, start building username
		// and group information
		username := ""
		groups := []string{}
		userExtra := map[string][]string{}
		uid := ""
		for _, impersonationRequest := range impersonationRequests {
			actingAsAttributes := &authorizer.AttributesRecord{
				User:            requestor,
				Verb:            "impersonate",
				Namespace:       impersonationRequest.Namespace,
				Name:            impersonationRequest.Name,
				ResourceRequest: true,
			}

			switch impersonationRequest.Kind {
			case "ServiceAccount":
				actingAsAttributes.Resource = "serviceaccounts"
				username = serviceaccount.MakeUsername(impersonationRequest.Namespace, impersonationRequest.Name)
				if !groupsSpecified {
					// if groups aren't specified for a service account, we know the groups because its a fixed mapping.  Add them
					groups = serviceaccount.MakeGroupNames(impersonationRequest.Namespace)
				}

			case "User":
				actingAsAttributes.Resource = "users"
				username = impersonationRequest.Name

			case "Group":
				actingAsAttributes.Resource = "groups"
				groups = append(groups, impersonationRequest.Name)

			case "UserExtra":
				extraKey := impersonationRequest.FieldPath
				extraValue := impersonationRequest.Name
				actingAsAttributes.Resource = "userextras"
				actingAsAttributes.Subresource = extraKey
				userExtra[extraKey] = append(userExtra[extraKey], extraValue)
			case "UID":
				uid = string(impersonationRequest.Name)
				actingAsAttributes.Resource = "uids"
			default:
				klog.V(4).InfoS("unknown impersonation request type", "Request", impersonationRequest)
				responsewriters.Forbidden(ctx, actingAsAttributes, w, req, fmt.Sprintf("unknown impersonation request type: %v", impersonationRequest), s)
				return
			}

			if a != nil {
				decision, reason, err := a.Authorize(ctx, actingAsAttributes)
				if err != nil || decision != authorizer.DecisionAllow {
					klog.V(4).InfoS("Forbidden", "URI", req.RequestURI, "Reason", reason, "Error", err)
					responsewriters.Forbidden(ctx, actingAsAttributes, w, req, reason, s)
					return
				}
			}
		}

		if username != user.Anonymous {
			// When impersonating a non-anonymous user, include the 'system:authenticated' group
			// in the impersonated user info:
			// - if no groups were specified
			// - if a group has been specified other than 'system:authenticated'
			//
			// If 'system:unauthenticated' group has been specified we should not include
			// the 'system:authenticated' group.
			addAuthenticated := true
			for _, group := range groups {
				if group == user.AllAuthenticated || group == user.AllUnauthenticated {
					addAuthenticated = false
					break
				}
			}

			if addAuthenticated {
				groups = append(groups, user.AllAuthenticated)
			}
		} else {
			addUnauthenticated := true
			for _, group := range groups {
				if group == user.AllUnauthenticated {
					addUnauthenticated = false
					break
				}
			}

			if addUnauthenticated {
				groups = append(groups, user.AllUnauthenticated)
			}
		}

		newUser := &user.DefaultInfo{
			Name:   username,
			Groups: groups,
			Extra:  userExtra,
			UID:    uid,
		}
		req = req.WithContext(request.WithUser(ctx, newUser))

		oldUser, _ := request.UserFrom(ctx)
		httplog.LogOf(req, w).Addf("%v is acting as %v", oldUser, newUser)

		ae := audit.AuditEventFrom(ctx)
		audit.LogImpersonatedUser(ae, newUser)

		// clear all the impersonation headers from the request
		req.Header.Del(api.ImpersonateUserHeader)
		req.Header.Del(api.ImpersonateGroupHeader)
		req.Header.Del(api.ImpersonateUIDHeader)
		for headerName := range req.Header {
			if strings.HasPrefix(headerName, api.ImpersonateUserExtraHeaderPrefix) {
				req.Header.Del(headerName)
			}
		}

		handler.ServeHTTP(w, req)
	})
}

func unescapeExtraKey(encodedKey string) string {
	key, err := url.PathUnescape(encodedKey) // Decode %-encoded bytes.
	if err != nil {
		return encodedKey // Always record extra strings, even if malformed/unencoded.
	}
	return key
}

// buildImpersonationRequests returns a list of objectreferences that represent the different things we're requesting to impersonate.
// Also includes a map[string][]string representing user.Info.Extra
// Each request must be authorized against the current user before switching contexts.
func buildImpersonationRequests(headers http.Header) ([]api.ObjectReference, error) {
	impersonationRequests := []api.ObjectReference{}

	requestedUser := headers.Get(api.ImpersonateUserHeader)
	hasUser := len(requestedUser) > 0
	if hasUser {
		if namespace, name, err := serviceaccount.SplitUsername(requestedUser); err == nil {
			impersonationRequests = append(impersonationRequests, api.ObjectReference{Kind: "ServiceAccount", Namespace: namespace, Name: name})
		} else {
			impersonationRequests = append(impersonationRequests, api.ObjectReference{Kind: "User", Name: requestedUser})
		}
	}

	hasGroups := false
	for _, group := range headers[api.ImpersonateGroupHeader] {
		hasGroups = true
		impersonationRequests = append(impersonationRequests, api.ObjectReference{Kind: "Group", Name: group})
	}

	hasUserExtra := false
	for headerName, values := range headers {
		if !strings.HasPrefix(headerName, api.ImpersonateUserExtraHeaderPrefix) {
			continue
		}

		hasUserExtra = true
		extraKey := unescapeExtraKey(strings.ToLower(headerName[len(api.ImpersonateUserExtraHeaderPrefix):]))

		// make a separate request for each extra value they're trying to set
		for _, value := range values {
			impersonationRequests = append(impersonationRequests,
				api.ObjectReference{
					Kind: "UserExtra",
					Name: value,
					// ObjectReference doesn't have a subresource field.  FieldPath is close and available, so we'll use that
					// TODO fight the good fight for ObjectReference to refer to resources and subresources
					FieldPath: extraKey,
				})
		}
	}

	requestedUID := headers.Get(api.ImpersonateUIDHeader)
	hasUID := len(requestedUID) > 0
	if hasUID {
		impersonationRequests = append(impersonationRequests, api.ObjectReference{
			Kind: "UID",
			Name: requestedUID,
		})
	}

	if (hasGroups || hasUserExtra || hasUID) && !hasUser {
		return nil, fmt.Errorf("requested %v without impersonating a user", impersonationRequests)
	}

	return impersonationRequests, nil
}
