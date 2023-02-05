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

// Package rbac implements the authorizer.Authorizer interface using roles base access control.
package rbac

import (
	"bytes"
	"context"
	"fmt"

	"k8s.io/klog/v2"

	"github.com/yubo/apiserver/pkg/apis/rbac"
	"github.com/yubo/apiserver/pkg/authentication/user"
	"github.com/yubo/apiserver/pkg/authorization/authorizer"
	"github.com/yubo/apiserver/pkg/listers"
	rbacregistryvalidation "github.com/yubo/apiserver/plugin/authorizer/rbac/validation"
	"github.com/yubo/golib/api"
	utilerrors "github.com/yubo/golib/util/errors"
)

type RequestToRuleMapper interface {
	// RulesFor returns all known PolicyRules and any errors that happened while locating those rules.
	// Any rule returned is still valid, since rules are deny by default.  If you can pass with the rules
	// supplied, you do not have to fail the request.  If you cannot, you should indicate the error along
	// with your denial.
	RulesFor(subject user.Info, namespace string) ([]rbac.PolicyRule, error)

	// VisitRulesFor invokes visitor() with each rule that applies to a given user in a given namespace,
	// and each error encountered resolving those rules. Rule may be nil if err is non-nil.
	// If visitor() returns false, visiting is short-circuited.
	VisitRulesFor(user user.Info, namespace string, visitor func(source fmt.Stringer, rule *rbac.PolicyRule, err error) bool)
}

type RBACAuthorizer struct {
	authorizationRuleResolver RequestToRuleMapper
}

// authorizingVisitor short-circuits once allowed, and collects any resolution errors encountered
type authorizingVisitor struct {
	requestAttributes authorizer.Attributes

	allowed bool
	reason  string
	errors  []error
}

func (v *authorizingVisitor) visit(source fmt.Stringer, rule *rbac.PolicyRule, err error) bool {
	if rule != nil && RuleAllows(v.requestAttributes, rule) {
		v.allowed = true
		v.reason = fmt.Sprintf("RBAC: allowed by %s", source.String())
		return false
	}
	if err != nil {
		v.errors = append(v.errors, err)
	}
	return true
}

func (r *RBACAuthorizer) Authorize(ctx context.Context, requestAttributes authorizer.Attributes) (authorizer.Decision, string, error) {
	klog.V(5).InfoS("authz.rbac",
		"user", requestAttributes.GetUser(),
		"verb", requestAttributes.GetVerb(),
		"isReadOnly", requestAttributes.IsReadOnly(),
		"namespace", requestAttributes.GetNamespace(),
		"resource", requestAttributes.GetResource(),
		"subResource", requestAttributes.GetSubresource(),
		"name", requestAttributes.GetName(),
		"isResourceRequest", requestAttributes.IsResourceRequest(),
		"path", requestAttributes.GetPath(),
	)

	ruleCheckingVisitor := &authorizingVisitor{requestAttributes: requestAttributes}

	r.authorizationRuleResolver.VisitRulesFor(requestAttributes.GetUser(), requestAttributes.GetNamespace(), ruleCheckingVisitor.visit)
	if ruleCheckingVisitor.allowed {
		return authorizer.DecisionAllow, ruleCheckingVisitor.reason, nil
	}

	// Build a detailed log of the denial.
	// Make the whole block conditional so we don't do a lot of string-building we won't use.
	if klog.V(5).Enabled() {
		var operation string
		if requestAttributes.IsResourceRequest() {
			b := &bytes.Buffer{}
			b.WriteString(`"`)
			b.WriteString(requestAttributes.GetVerb())
			b.WriteString(`" resource "`)
			b.WriteString(requestAttributes.GetResource())
			if len(requestAttributes.GetAPIGroup()) > 0 {
				b.WriteString(`.`)
				b.WriteString(requestAttributes.GetAPIGroup())
			}
			if len(requestAttributes.GetSubresource()) > 0 {
				b.WriteString(`/`)
				b.WriteString(requestAttributes.GetSubresource())
			}
			b.WriteString(`"`)
			if len(requestAttributes.GetName()) > 0 {
				b.WriteString(` named "`)
				b.WriteString(requestAttributes.GetName())
				b.WriteString(`"`)
			}
			operation = b.String()
		} else {
			operation = fmt.Sprintf("%q nonResourceURL %q", requestAttributes.GetVerb(), requestAttributes.GetPath())
		}

		var scope string
		if ns := requestAttributes.GetNamespace(); len(ns) > 0 {
			scope = fmt.Sprintf("in namespace %q", ns)
		} else {
			scope = "cluster-wide"
		}

		klog.Infof("RBAC: no rules authorize user %q with groups %q to %s %s", requestAttributes.GetUser().GetName(), requestAttributes.GetUser().GetGroups(), operation, scope)
	}

	reason := ""
	if len(ruleCheckingVisitor.errors) > 0 {
		reason = fmt.Sprintf("RBAC: %v", utilerrors.NewAggregate(ruleCheckingVisitor.errors))
	}
	return authorizer.DecisionNoOpinion, reason, nil
}

func (r *RBACAuthorizer) RulesFor(user user.Info, namespace string) ([]authorizer.ResourceRuleInfo, []authorizer.NonResourceRuleInfo, bool, error) {
	var (
		resourceRules    []authorizer.ResourceRuleInfo
		nonResourceRules []authorizer.NonResourceRuleInfo
	)

	policyRules, err := r.authorizationRuleResolver.RulesFor(user, namespace)
	for _, policyRule := range policyRules {
		if len(policyRule.Resources) > 0 {
			r := authorizer.DefaultResourceRuleInfo{
				Verbs:         policyRule.Verbs,
				APIGroups:     policyRule.APIGroups,
				Resources:     policyRule.Resources,
				ResourceNames: policyRule.ResourceNames,
			}
			var resourceRule authorizer.ResourceRuleInfo = &r
			resourceRules = append(resourceRules, resourceRule)
		}
		if len(policyRule.NonResourceURLs) > 0 {
			r := authorizer.DefaultNonResourceRuleInfo{
				Verbs:           policyRule.Verbs,
				NonResourceURLs: policyRule.NonResourceURLs,
			}
			var nonResourceRule authorizer.NonResourceRuleInfo = &r
			nonResourceRules = append(nonResourceRules, nonResourceRule)
		}
	}
	return resourceRules, nonResourceRules, false, err
}

func New(roles rbacregistryvalidation.RoleGetter, roleBindings rbacregistryvalidation.RoleBindingLister, clusterRoles rbacregistryvalidation.ClusterRoleGetter, clusterRoleBindings rbacregistryvalidation.ClusterRoleBindingLister) *RBACAuthorizer {
	authorizer := &RBACAuthorizer{
		authorizationRuleResolver: rbacregistryvalidation.NewDefaultRuleResolver(
			roles, roleBindings, clusterRoles, clusterRoleBindings,
		),
	}
	return authorizer
}

func RulesAllow(requestAttributes authorizer.Attributes, rules ...rbac.PolicyRule) bool {
	klog.InfoS("rules allows")
	for i := range rules {
		if RuleAllows(requestAttributes, &rules[i]) {
			return true
		}
	}

	return false
}

func RuleAllows(requestAttributes authorizer.Attributes, rule *rbac.PolicyRule) bool {
	klog.InfoS("rule allows", "requestAttributes.IsResourceRequest", requestAttributes.IsResourceRequest())
	if requestAttributes.IsResourceRequest() {
		combinedResource := requestAttributes.GetResource()
		if len(requestAttributes.GetSubresource()) > 0 {
			combinedResource = requestAttributes.GetResource() + "/" + requestAttributes.GetSubresource()
		}

		return rbac.VerbMatches(rule, requestAttributes.GetVerb()) &&
			//rbac.APIGroupMatches(rule, requestAttributes.GetAPIGroup()) &&
			rbac.ResourceMatches(rule, combinedResource, requestAttributes.GetSubresource()) &&
			rbac.ResourceNameMatches(rule, requestAttributes.GetName())
	}

	klog.InfoS("rule allows", "verbmatches", rbac.VerbMatches(rule, requestAttributes.GetVerb()), "nonresource matches", rbac.NonResourceURLMatches(rule, requestAttributes.GetPath()))
	return rbac.VerbMatches(rule, requestAttributes.GetVerb()) &&
		rbac.NonResourceURLMatches(rule, requestAttributes.GetPath())
}

type RoleGetter struct {
	Lister listers.RoleLister
}

func (g *RoleGetter) GetRole(namespace, name string) (*rbac.Role, error) {
	return g.Lister.Get(context.TODO(), name)
}

type RoleBindingLister struct {
	Lister listers.RoleBindingLister
}

func (l *RoleBindingLister) ListRoleBindings(namespace string) ([]*rbac.RoleBinding, error) {
	if namespace != "" {
	}
	list, err := l.Lister.List(context.TODO(), api.GetListOptions{})
	return list, err
}

type ClusterRoleGetter struct {
	Lister listers.ClusterRoleLister
}

func (g *ClusterRoleGetter) GetClusterRole(name string) (*rbac.ClusterRole, error) {
	return g.Lister.Get(context.TODO(), name)
}

type ClusterRoleBindingLister struct {
	Lister listers.ClusterRoleBindingLister
}

func (l *ClusterRoleBindingLister) ListClusterRoleBindings() ([]*rbac.ClusterRoleBinding, error) {
	list, err := l.Lister.List(context.TODO(), api.GetListOptions{})
	return list, err
}
