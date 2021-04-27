package authorization

import (
	"context"
	"fmt"
	"time"

	"github.com/yubo/apiserver/pkg/authorization/abac"
	"github.com/yubo/apiserver/pkg/authorization/authorizer"
	"github.com/yubo/apiserver/pkg/authorization/authorizerfactory"
	authzmodes "github.com/yubo/apiserver/pkg/authorization/modes"
	"github.com/yubo/apiserver/pkg/authorization/rbac"
	"github.com/yubo/apiserver/pkg/authorization/union"
	"github.com/yubo/apiserver/pkg/listers"
	"github.com/yubo/apiserver/pkg/options"
	utilnet "github.com/yubo/golib/staging/util/net"
	"github.com/yubo/golib/staging/util/wait"
	"k8s.io/klog/v2"
)

// Config provides an easy way for composing API servers to delegate their authorization to
// the root kube API server.
// WARNING: never assume that every authenticated incoming request already does authorization.
//          The aggregator in the kube API server does this today, but this behaviour is not
//          guaranteed in the future.
type Authorization struct {
	AuthorizationModes []string

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

	// Optional field, custom dial function used to connect to webhook
	CustomDial utilnet.DialFunc

	// Authorizer determines whether the subject is allowed to make the request based only
	// on the RequestURI
	Authorizer   authorizer.Authorizer
	RuleResolver authorizer.RuleResolver
}

func newAuthorization(ctx context.Context, o *config) (authz *Authorization, err error) {
	authz = &Authorization{
		AuthorizationModes:          o.Modes,
		PolicyFile:                  o.PolicyFile,
		WebhookConfigFile:           o.WebhookConfigFile,
		WebhookVersion:              o.WebhookVersion,
		WebhookCacheAuthorizedTTL:   o.WebhookCacheAuthorizedTTL,
		WebhookCacheUnauthorizedTTL: o.WebhookCacheUnauthorizedTTL,
		//VersionedInformerFactory:    versionedInformerFactory,
		WebhookRetryBackoff: o.WebhookRetryBackoff,
	}

	authz.Authorizer, authz.RuleResolver, err = authz.New(ctx)
	if err != nil {
		return nil, err
	}

	return authz, nil
}

// New returns the right sort of union of multiple authorizer.Authorizer objects
// based on the authorizationMode or an error.
func (authz Authorization) New(ctx context.Context) (authorizer.Authorizer, authorizer.RuleResolver, error) {
	klog.V(5).Infof("authz %+v", authz.AuthorizationModes)
	if len(authz.AuthorizationModes) == 0 {
		return nil, nil, fmt.Errorf("at least one authorization mode must be passed")
	}

	var (
		authorizers   []authorizer.Authorizer
		ruleResolvers []authorizer.RuleResolver
	)

	for _, authorizationMode := range authz.AuthorizationModes {
		// Keep cases in sync with constant list in github.com/yubo/apiserver/pkg/authorization/modes/modes.go.
		switch authorizationMode {
		case authzmodes.ModeNode:
			/*
				node.RegisterMetrics()
				graph := node.NewGraph()
				node.AddGraphEventHandlers(
					graph,
					authz.VersionedInformerFactory.Core().V1().Nodes(),
					authz.VersionedInformerFactory.Core().V1().Pods(),
					authz.VersionedInformerFactory.Core().V1().PersistentVolumes(),
					authz.VersionedInformerFactory.Storage().V1().VolumeAttachments(),
				)
				nodeAuthorizer := node.NewAuthorizer(graph, nodeidentifier.NewDefaultNodeIdentifier(), bootstrappolicy.NodeRules())
				authorizers = append(authorizers, nodeAuthorizer)
				ruleResolvers = append(ruleResolvers, nodeAuthorizer)
			*/

		case authzmodes.ModeAlwaysAllow:
			alwaysAllowAuthorizer := authorizerfactory.NewAlwaysAllowAuthorizer()
			authorizers = append(authorizers, alwaysAllowAuthorizer)
			ruleResolvers = append(ruleResolvers, alwaysAllowAuthorizer)
		case authzmodes.ModeAlwaysDeny:
			alwaysDenyAuthorizer := authorizerfactory.NewAlwaysDenyAuthorizer()
			authorizers = append(authorizers, alwaysDenyAuthorizer)
			ruleResolvers = append(ruleResolvers, alwaysDenyAuthorizer)
		case authzmodes.ModeABAC:
			abacAuthorizer, err := abac.NewFromFile(authz.PolicyFile)
			if err != nil {
				return nil, nil, err
			}
			authorizers = append(authorizers, abacAuthorizer)
			ruleResolvers = append(ruleResolvers, abacAuthorizer)
		//case authzmodes.ModeWebhook:
		//	if authz.WebhookRetryBackoff == nil {
		//		return nil, nil, errors.New("retry backoff parameters for authorization webhook has not been specified")
		//	}
		//	webhookAuthorizer, err := webhook.New(authz.WebhookConfigFile,
		//		authz.WebhookVersion,
		//		authz.WebhookCacheAuthorizedTTL,
		//		authz.WebhookCacheUnauthorizedTTL,
		//		*authz.WebhookRetryBackoff,
		//		authz.CustomDial)
		//	if err != nil {
		//		return nil, nil, err
		//	}
		//	authorizers = append(authorizers, webhookAuthorizer)
		//	ruleResolvers = append(ruleResolvers, webhookAuthorizer)
		case authzmodes.ModeRBAC:
			// YUBO TODO
			db, ok := options.DBFrom(ctx)
			if !ok {
				return nil, nil, fmt.Errorf("unable to get db provider")
			}
			rbacAuthorizer := rbac.New(
				&rbac.RoleGetter{Lister: listers.NewRoleLister(db)},
				&rbac.RoleBindingLister{Lister: listers.NewRoleBindingLister(db)},
				&rbac.ClusterRoleGetter{Lister: listers.NewClusterRoleLister(db)},
				&rbac.ClusterRoleBindingLister{Lister: listers.NewClusterRoleBindingLister(db)},
			)
			authorizers = append(authorizers, rbacAuthorizer)
			ruleResolvers = append(ruleResolvers, rbacAuthorizer)
		default:
			return nil, nil, fmt.Errorf("unknown authorization mode %s specified", authorizationMode)
		}
	}

	return union.New(authorizers...), union.NewRuleResolvers(ruleResolvers...), nil
}
