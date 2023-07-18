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

// package authorizer
package authorizer

import (
	"errors"
	"fmt"
	"time"

	"github.com/yubo/apiserver/pkg/authentication/user"
	"github.com/yubo/apiserver/pkg/authorization/authorizer"
	"github.com/yubo/apiserver/pkg/authorization/authorizer/modes"
	"github.com/yubo/apiserver/pkg/authorization/authorizerfactory"
	"github.com/yubo/apiserver/pkg/authorization/path"
	"github.com/yubo/apiserver/pkg/authorization/union"
	webhookutil "github.com/yubo/apiserver/pkg/util/webhook"
	"github.com/yubo/apiserver/plugin/authorizer/abac"
	"github.com/yubo/apiserver/plugin/authorizer/rbac/db"
	"github.com/yubo/apiserver/plugin/authorizer/rbac/file"
	"github.com/yubo/apiserver/plugin/authorizer/webhook"
	utilnet "github.com/yubo/golib/util/net"
	"github.com/yubo/golib/util/wait"
)

// Config contains the data on how to authorize a request to the Kube API Server
type Config struct {
	AuthorizationModes []string

	// Options for ModeRBAC

	// Path to an RBAC config dir.
	RBACConfigDir string

	// Options for ModeABAC

	// Path to an ABAC policy file.
	PolicyFile string

	// Options for ModeWebhook

	// Kubeconfig file for Webhook authorization plugin.
	WebhookConfigFile string
	// API version of subject access reviews to send to the webhook (e.g. "v1", "v1beta1")
	WebhookVersion string
	// TTL for caching of authorized responses from the webhook server.
	WebhookCacheAuthorizedTTL time.Duration
	// TTL for caching of unauthorized responses from the webhook server.
	WebhookCacheUnauthorizedTTL time.Duration
	// WebhookRetryBackoff specifies the backoff parameters for the authorization webhook retry logic.
	// This allows us to configure the sleep time at each iteration and the maximum number of retries allowed
	// before we fail the webhook call in order to limit the fan out that ensues when the system is degraded.
	WebhookRetryBackoff *wait.Backoff

	//VersionedInformerFactory versionedinformers.SharedInformerFactory

	// Optional field, custom dial function used to connect to webhook
	CustomDial utilnet.DialFunc

	// AlwaysAllowPaths are HTTP paths which are excluded from authorization. They can be plain
	// paths or end in * in which case prefix-match is applied. A leading / is optional.
	AlwaysAllowPaths []string

	// AlwaysAllowGroups are groups which are allowed to take any actions.  In kube, this is system:masters.
	AlwaysAllowGroups []string
}

// New returns the right sort of union of multiple authorizer.Authorizer objects
// based on the authorizationMode or an error.
func (config Config) New() (authorizer.Authorizer, authorizer.RuleResolver, error) {
	if len(config.AuthorizationModes) == 0 {
		return nil, nil, fmt.Errorf("at least one authorization mode must be passed")
	}

	var (
		authorizers   []authorizer.Authorizer
		ruleResolvers []authorizer.RuleResolver
	)

	// Add SystemPrivilegedGroup as an authorizing group
	if len(config.AlwaysAllowGroups) > 0 {
		authorizers = append(authorizers, authorizerfactory.NewPrivilegedGroups(config.AlwaysAllowGroups...))
	} else {
		authorizers = append(authorizers, authorizerfactory.NewPrivilegedGroups(user.SystemPrivilegedGroup))
	}

	// add AlwaysAllowPaths
	if len(config.AlwaysAllowPaths) > 0 {
		a, err := path.NewAuthorizer(config.AlwaysAllowPaths)
		if err != nil {
			return nil, nil, err
		}
		authorizers = append(authorizers, a)
	}

	for _, authorizationMode := range config.AuthorizationModes {
		// Keep cases in sync with constant list in k8s.io/kubernetes/pkg/kubeapiserver/authorizer/modes/modes.go.
		switch authorizationMode {
		//case modes.ModeNode:
		//	node.RegisterMetrics()
		//	graph := node.NewGraph()
		//	node.AddGraphEventHandlers(
		//		graph,
		//		config.VersionedInformerFactory.Core().V1().Nodes(),
		//		config.VersionedInformerFactory.Core().V1().Pods(),
		//		config.VersionedInformerFactory.Core().V1().PersistentVolumes(),
		//		config.VersionedInformerFactory.Storage().V1().VolumeAttachments(),
		//	)
		//	nodeAuthorizer := node.NewAuthorizer(graph, nodeidentifier.NewDefaultNodeIdentifier(), bootstrappolicy.NodeRules())
		//	authorizers = append(authorizers, nodeAuthorizer)
		//	ruleResolvers = append(ruleResolvers, nodeAuthorizer)

		case modes.ModeAlwaysAllow:
			alwaysAllowAuthorizer := authorizerfactory.NewAlwaysAllowAuthorizer()
			authorizers = append(authorizers, alwaysAllowAuthorizer)
			ruleResolvers = append(ruleResolvers, alwaysAllowAuthorizer)
		case modes.ModeAlwaysDeny:
			alwaysDenyAuthorizer := authorizerfactory.NewAlwaysDenyAuthorizer()
			authorizers = append(authorizers, alwaysDenyAuthorizer)
			ruleResolvers = append(ruleResolvers, alwaysDenyAuthorizer)
		case modes.ModeABAC:
			abacAuthorizer, err := abac.NewFromFile(config.PolicyFile)
			if err != nil {
				return nil, nil, err
			}
			authorizers = append(authorizers, abacAuthorizer)
			ruleResolvers = append(ruleResolvers, abacAuthorizer)
		case modes.ModeWebhook:
			if config.WebhookRetryBackoff == nil {
				return nil, nil, errors.New("retry backoff parameters for authorization webhook has not been specified")
			}
			clientConfig, err := webhookutil.LoadKubeconfig(config.WebhookConfigFile, config.CustomDial)
			if err != nil {
				return nil, nil, err
			}
			webhookAuthorizer, err := webhook.New(clientConfig,
				config.WebhookVersion,
				config.WebhookCacheAuthorizedTTL,
				config.WebhookCacheUnauthorizedTTL,
				*config.WebhookRetryBackoff,
			)
			if err != nil {
				return nil, nil, err
			}
			authorizers = append(authorizers, webhookAuthorizer)
			ruleResolvers = append(ruleResolvers, webhookAuthorizer)
		case modes.ModeRBAC:
			if config.RBACConfigDir != "" {
				rbacAuthorizer, err := file.NewRBAC(config.RBACConfigDir)
				if err != nil {
					return nil, nil, err
				}
				authorizers = append(authorizers, rbacAuthorizer)
				ruleResolvers = append(ruleResolvers, rbacAuthorizer)

			} else {
				rbacAuthorizer, err := db.NewRBAC()
				if err != nil {
					return nil, nil, err
				}
				authorizers = append(authorizers, rbacAuthorizer)
				ruleResolvers = append(ruleResolvers, rbacAuthorizer)
			}
		default:
			return nil, nil, fmt.Errorf("unknown authorization mode %s specified", authorizationMode)
		}
	}

	return union.New(authorizers...), union.NewRuleResolvers(ruleResolvers...), nil
}
