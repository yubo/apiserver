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

// Package webhook implements the authorizer.Authorizer interface using HTTP webhooks.
package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime/debug"
	"time"

	"github.com/yubo/apiserver/pkg/apis/authorization"
	"github.com/yubo/apiserver/pkg/authentication/user"
	"github.com/yubo/apiserver/pkg/authorization/authorizer"
	"github.com/yubo/apiserver/pkg/util/webhook"
	"github.com/yubo/golib/api"
	"github.com/yubo/golib/scheme"
	"github.com/yubo/golib/util/cache"
	utilnet "github.com/yubo/golib/util/net"
	"github.com/yubo/golib/util/wait"
	"k8s.io/klog/v2"
)

const (
	// The maximum length of requester-controlled attributes to allow caching.
	maxControlledAttrCacheSize = 10000
)

// DefaultRetryBackoff returns the default backoff parameters for webhook retry.
func DefaultRetryBackoff() *wait.Backoff {
	backoff := webhook.DefaultRetryBackoffWithInitialDelay(500 * time.Millisecond)
	return &backoff
}

// Ensure Webhook implements the authorizer.Authorizer interface.
var _ authorizer.Authorizer = (*WebhookAuthorizer)(nil)

type subjectAccessReviewer interface {
	Create(context.Context, *authorization.SubjectAccessReview, api.CreateOptions) (*authorization.SubjectAccessReview, error)
}

type WebhookAuthorizer struct {
	subjectAccessReview subjectAccessReviewer
	responseCache       *cache.LRUExpireCache
	authorizedTTL       time.Duration
	unauthorizedTTL     time.Duration
	retryBackoff        wait.Backoff
	decisionOnError     authorizer.Decision
}

// NewFromInterface creates a WebhookAuthorizer using the given subjectAccessReview client
func NewFromInterface(subjectAccessReview authorization.SubjectAccessReviewInterface, authorizedTTL, unauthorizedTTL time.Duration, retryBackoff wait.Backoff) (*WebhookAuthorizer, error) {
	return newWithBackoff(subjectAccessReview, authorizedTTL, unauthorizedTTL, retryBackoff)
}

// New creates a new WebhookAuthorizer from the provided kubeconfig file.
// The config's cluster field is used to refer to the remote service, user refers to the returned authorizer.
//
//     # clusters refers to the remote service.
//     clusters:
//     - name: name-of-remote-authz-service
//       cluster:
//         certificate-authority: /path/to/ca.pem      # CA for verifying the remote service.
//         server: https://authz.example.com/authorize # URL of remote service to query. Must use 'https'.
//
//     # users refers to the API server's webhook configuration.
//     users:
//     - name: name-of-api-server
//       user:
//         client-certificate: /path/to/cert.pem # cert for the webhook plugin to use
//         client-key: /path/to/key.pem          # key matching the cert
//
// For additional HTTP configuration, refer to the kubeconfig documentation
// https://kubernetes.io/docs/user-guide/kubeconfig-file/.
func New(kubeConfigFile string, version string, authorizedTTL, unauthorizedTTL time.Duration, retryBackoff wait.Backoff, customDial utilnet.DialFunc) (*WebhookAuthorizer, error) {
	subjectAccessReview, err := subjectAccessReviewInterfaceFromKubeconfig(kubeConfigFile, retryBackoff, customDial)
	if err != nil {
		return nil, err
	}
	return newWithBackoff(subjectAccessReview, authorizedTTL, unauthorizedTTL, retryBackoff)
}

// newWithBackoff allows tests to skip the sleep.
func newWithBackoff(subjectAccessReview subjectAccessReviewer, authorizedTTL, unauthorizedTTL time.Duration, retryBackoff wait.Backoff) (*WebhookAuthorizer, error) {
	return &WebhookAuthorizer{
		subjectAccessReview: subjectAccessReview,
		responseCache:       cache.NewLRUExpireCache(8192),
		authorizedTTL:       authorizedTTL,
		unauthorizedTTL:     unauthorizedTTL,
		retryBackoff:        retryBackoff,
		decisionOnError:     authorizer.DecisionNoOpinion,
	}, nil
}

// Authorize makes a REST request to the remote service describing the attempted action as a JSON
// serialized api.authorization.v1beta1.SubjectAccessReview object. An example request body is
// provided below.
//
//     {
//       "apiVersion": "authorization.k8s.io/v1beta1",
//       "kind": "SubjectAccessReview",
//       "spec": {
//         "resourceAttributes": {
//           "namespace": "kittensandponies",
//           "verb": "GET",
//           "group": "group3",
//           "resource": "pods"
//         },
//         "user": "jane",
//         "group": [
//           "group1",
//           "group2"
//         ]
//       }
//     }
//
// The remote service is expected to fill the SubjectAccessReviewStatus field to either allow or
// disallow access. A permissive response would return:
//
//     {
//       "apiVersion": "authorization.k8s.io/v1beta1",
//       "kind": "SubjectAccessReview",
//       "status": {
//         "allowed": true
//       }
//     }
//
// To disallow access, the remote service would return:
//
//     {
//       "apiVersion": "authorization.k8s.io/v1beta1",
//       "kind": "SubjectAccessReview",
//       "status": {
//         "allowed": false,
//         "reason": "user does not have read access to the namespace"
//       }
//     }
//
// TODO(mikedanese): We should eventually support failing closed when we
// encounter an error. We are failing open now to preserve backwards compatible
// behavior.
func (w *WebhookAuthorizer) Authorize(ctx context.Context, attr authorizer.Attributes) (decision authorizer.Decision, reason string, err error) {
	r := &authorization.SubjectAccessReview{}
	if user := attr.GetUser(); user != nil {
		r.Spec = authorization.SubjectAccessReviewSpec{
			User:   user.GetName(),
			UID:    user.GetUID(),
			Groups: user.GetGroups(),
			Extra:  convertToSARExtra(user.GetExtra()),
		}
	}

	if attr.IsResourceRequest() {
		r.Spec.ResourceAttributes = &authorization.ResourceAttributes{
			Namespace:   attr.GetNamespace(),
			Verb:        attr.GetVerb(),
			Group:       attr.GetAPIGroup(),
			Version:     attr.GetAPIVersion(),
			Resource:    attr.GetResource(),
			Subresource: attr.GetSubresource(),
			Name:        attr.GetName(),
		}
	} else {
		r.Spec.NonResourceAttributes = &authorization.NonResourceAttributes{
			Path: attr.GetPath(),
			Verb: attr.GetVerb(),
		}
	}
	key, err := json.Marshal(r.Spec)
	if err != nil {
		return w.decisionOnError, "", err
	}
	if entry, ok := w.responseCache.Get(string(key)); ok {
		r.Status = entry.(authorization.SubjectAccessReviewStatus)
	} else {
		var result *authorization.SubjectAccessReview
		// WithExponentialBackoff will return SAR create error (sarErr) if any.
		if err := webhook.WithExponentialBackoff(ctx, w.retryBackoff, func() error {
			var sarErr error
			result, sarErr = w.subjectAccessReview.Create(ctx, r, api.CreateOptions{})
			return sarErr
		}, webhook.DefaultShouldRetry); err != nil {
			klog.Errorf("Failed to make webhook authorizer request: %v", err)
			debug.PrintStack()
			return w.decisionOnError, "", err
		}

		r.Status = result.Status
		if shouldCache(attr) {
			if r.Status.Allowed {
				w.responseCache.Add(string(key), r.Status, w.authorizedTTL)
			} else {
				w.responseCache.Add(string(key), r.Status, w.unauthorizedTTL)
			}
		}
	}
	switch {
	case r.Status.Denied && r.Status.Allowed:
		return authorizer.DecisionDeny, r.Status.Reason, fmt.Errorf("webhook subject access review returned both allow and deny response")
	case r.Status.Denied:
		return authorizer.DecisionDeny, r.Status.Reason, nil
	case r.Status.Allowed:
		return authorizer.DecisionAllow, r.Status.Reason, nil
	default:
		return authorizer.DecisionNoOpinion, r.Status.Reason, nil
	}

}

//TODO: need to finish the method to get the rules when using webhook mode
func (w *WebhookAuthorizer) RulesFor(user user.Info, namespace string) ([]authorizer.ResourceRuleInfo, []authorizer.NonResourceRuleInfo, bool, error) {
	var (
		resourceRules    []authorizer.ResourceRuleInfo
		nonResourceRules []authorizer.NonResourceRuleInfo
	)
	incomplete := true
	return resourceRules, nonResourceRules, incomplete, fmt.Errorf("webhook authorizer does not support user rule resolution")
}

func convertToSARExtra(extra map[string][]string) map[string]authorization.ExtraValue {
	if extra == nil {
		return nil
	}
	ret := map[string]authorization.ExtraValue{}
	for k, v := range extra {
		ret[k] = authorization.ExtraValue(v)
	}

	return ret
}

// subjectAccessReviewInterfaceFromKubeconfig builds a client from the specified kubeconfig file,
// and returns a SubjectAccessReviewInterface that uses that client. Note that the client submits SubjectAccessReview
// requests to the exact path specified in the kubeconfig file, so arbitrary non-API servers can be targeted.
func subjectAccessReviewInterfaceFromKubeconfig(kubeConfigFile string, retryBackoff wait.Backoff, customDial utilnet.DialFunc) (subjectAccessReviewer, error) {
	gw, err := webhook.NewGenericWebhook(scheme.Codecs, kubeConfigFile, retryBackoff, customDial)
	if err != nil {
		return nil, err
	}
	return &subjectAccessReviewV1Client{gw}, nil

}

type subjectAccessReviewV1Client struct {
	w *webhook.GenericWebhook
}

func (t *subjectAccessReviewV1Client) Create(ctx context.Context, subjectAccessReview *authorization.SubjectAccessReview, _ api.CreateOptions) (*authorization.SubjectAccessReview, error) {
	result := &authorization.SubjectAccessReview{}
	err := t.w.RestClient.Post().Body(subjectAccessReview).Do(ctx).Into(result)
	return result, err
}

// shouldCache determines whether it is safe to cache the given request attributes. If the
// requester-controlled attributes are too large, this may be a DoS attempt, so we skip the cache.
func shouldCache(attr authorizer.Attributes) bool {
	controlledAttrSize := int64(len(attr.GetNamespace())) +
		int64(len(attr.GetVerb())) +
		int64(len(attr.GetAPIGroup())) +
		int64(len(attr.GetAPIVersion())) +
		int64(len(attr.GetResource())) +
		int64(len(attr.GetSubresource())) +
		int64(len(attr.GetName())) +
		int64(len(attr.GetPath()))
	return controlledAttrSize < maxControlledAttrCacheSize
}
