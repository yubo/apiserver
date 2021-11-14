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

package validation

import (
	"errors"
	"fmt"

	"github.com/yubo/apiserver/pkg/authentication/serviceaccount"
	"github.com/yubo/apiserver/pkg/authentication/user"
	"github.com/yubo/apiserver/pkg/apis/rbac"
	utilerrors "github.com/yubo/golib/util/errors"
	//"k8s.io/component-helpers/auth/rbac/validation"
)

type AuthorizationRuleResolver interface {
	// GetRoleReferenceRules attempts to resolve the role reference of a RoleBinding or ClusterRoleBinding.  The passed namespace should be the namespace
	// of the role binding, the empty string if a cluster role binding.
	GetRoleReferenceRules(roleRef rbac.RoleRef, namespace string) ([]rbac.PolicyRule, error)

	// RulesFor returns the list of rules that apply to a given user in a given namespace and error.  If an error is returned, the slice of
	// PolicyRules may not be complete, but it contains all retrievable rules.  This is done because policy rules are purely additive and policy determinations
	// can be made on the basis of those rules that are found.
	RulesFor(user user.Info, namespace string) ([]rbac.PolicyRule, error)

	// VisitRulesFor invokes visitor() with each rule that applies to a given user in a given namespace, and each error encountered resolving those rules.
	// If visitor() returns false, visiting is short-circuited.
	VisitRulesFor(user user.Info, namespace string, visitor func(source fmt.Stringer, rule *rbac.PolicyRule, err error) bool)
}

// ConfirmNoEscalation determines if the roles for a given user in a given namespace encompass the provided role.
//func ConfirmNoEscalation(ctx context.Context, ruleResolver AuthorizationRuleResolver, rules []rbac.PolicyRule) error {
//	ruleResolutionErrors := []error{}
//
//	user, ok := genericapirequest.UserFrom(ctx)
//	if !ok {
//		return fmt.Errorf("no user on context")
//	}
//	namespace, _ := genericapirequest.NamespaceFrom(ctx)
//
//	ownerRules, err := ruleResolver.RulesFor(user, namespace)
//	if err != nil {
//		// As per AuthorizationRuleResolver contract, this may return a non fatal error with an incomplete list of policies. Log the error and continue.
//		klog.V(1).Infof("non-fatal error getting local rules for %v: %v", user, err)
//		ruleResolutionErrors = append(ruleResolutionErrors, err)
//	}
//
//	ownerRightsCover, missingRights := validation.Covers(ownerRules, rules)
//	if !ownerRightsCover {
//		compactMissingRights := missingRights
//		if compact, err := CompactRules(missingRights); err == nil {
//			compactMissingRights = compact
//		}
//
//		missingDescriptions := sets.NewString()
//		for _, missing := range compactMissingRights {
//			missingDescriptions.Insert(rbacv1helpers.CompactString(missing))
//		}
//
//		msg := fmt.Sprintf("user %q (groups=%q) is attempting to grant RBAC permissions not currently held:\n%s", user.GetName(), user.GetGroups(), strings.Join(missingDescriptions.List(), "\n"))
//		if len(ruleResolutionErrors) > 0 {
//			msg = msg + fmt.Sprintf("; resolution errors: %v", ruleResolutionErrors)
//		}
//
//		return errors.New(msg)
//	}
//	return nil
//}

type DefaultRuleResolver struct {
	roleGetter               RoleGetter
	roleBindingLister        RoleBindingLister
	clusterRoleGetter        ClusterRoleGetter
	clusterRoleBindingLister ClusterRoleBindingLister
}

func NewDefaultRuleResolver(roleGetter RoleGetter, roleBindingLister RoleBindingLister, clusterRoleGetter ClusterRoleGetter, clusterRoleBindingLister ClusterRoleBindingLister) *DefaultRuleResolver {
	return &DefaultRuleResolver{roleGetter, roleBindingLister, clusterRoleGetter, clusterRoleBindingLister}
}

type RoleGetter interface {
	GetRole(namespace, name string) (*rbac.Role, error)
}

type RoleBindingLister interface {
	ListRoleBindings(namespace string) ([]*rbac.RoleBinding, error)
}

type ClusterRoleGetter interface {
	GetClusterRole(name string) (*rbac.ClusterRole, error)
}

type ClusterRoleBindingLister interface {
	ListClusterRoleBindings() ([]*rbac.ClusterRoleBinding, error)
}

func (r *DefaultRuleResolver) RulesFor(user user.Info, namespace string) ([]rbac.PolicyRule, error) {
	visitor := &ruleAccumulator{}
	r.VisitRulesFor(user, namespace, visitor.visit)
	return visitor.rules, utilerrors.NewAggregate(visitor.errors)
}

type ruleAccumulator struct {
	rules  []rbac.PolicyRule
	errors []error
}

func (r *ruleAccumulator) visit(source fmt.Stringer, rule *rbac.PolicyRule, err error) bool {
	if rule != nil {
		r.rules = append(r.rules, *rule)
	}
	if err != nil {
		r.errors = append(r.errors, err)
	}
	return true
}

func describeSubject(s *rbac.Subject, bindingNamespace string) string {
	switch s.Kind {
	case rbac.ServiceAccountKind:
		if len(s.Namespace) > 0 {
			return fmt.Sprintf("%s %q", s.Kind, s.Name+"/"+s.Namespace)
		}
		return fmt.Sprintf("%s %q", s.Kind, s.Name+"/"+bindingNamespace)
	default:
		return fmt.Sprintf("%s %q", s.Kind, s.Name)
	}
}

type clusterRoleBindingDescriber struct {
	binding *rbac.ClusterRoleBinding
	subject *rbac.Subject
}

func (d *clusterRoleBindingDescriber) String() string {
	return fmt.Sprintf("ClusterRoleBinding %q of %s %q to %s",
		d.binding.Name,
		d.binding.RoleRef.Kind,
		d.binding.RoleRef.Name,
		describeSubject(d.subject, ""),
	)
}

type roleBindingDescriber struct {
	binding *rbac.RoleBinding
	subject *rbac.Subject
}

func (d *roleBindingDescriber) String() string {
	return fmt.Sprintf("RoleBinding %q of %s %q to %s",
		d.binding.Name+"/"+d.binding.Namespace,
		d.binding.RoleRef.Kind,
		d.binding.RoleRef.Name,
		describeSubject(d.subject, d.binding.Namespace),
	)
}

func (r *DefaultRuleResolver) VisitRulesFor(user user.Info, namespace string, visitor func(source fmt.Stringer, rule *rbac.PolicyRule, err error) bool) {
	if clusterRoleBindings, err := r.clusterRoleBindingLister.ListClusterRoleBindings(); err != nil {
		if !visitor(nil, nil, err) {
			return
		}
	} else {
		sourceDescriber := &clusterRoleBindingDescriber{}
		for _, clusterRoleBinding := range clusterRoleBindings {
			subjectIndex, applies := appliesTo(user, clusterRoleBinding.Subjects, "")
			if !applies {
				continue
			}
			rules, err := r.GetRoleReferenceRules(clusterRoleBinding.RoleRef, "")
			if err != nil {
				if !visitor(nil, nil, err) {
					return
				}
				continue
			}
			sourceDescriber.binding = clusterRoleBinding
			sourceDescriber.subject = &clusterRoleBinding.Subjects[subjectIndex]
			for i := range rules {
				if !visitor(sourceDescriber, &rules[i], nil) {
					return
				}
			}
		}
	}

	if len(namespace) > 0 {
		if roleBindings, err := r.roleBindingLister.ListRoleBindings(namespace); err != nil {
			if !visitor(nil, nil, err) {
				return
			}
		} else {
			sourceDescriber := &roleBindingDescriber{}
			for _, roleBinding := range roleBindings {
				subjectIndex, applies := appliesTo(user, roleBinding.Subjects, namespace)
				if !applies {
					continue
				}
				rules, err := r.GetRoleReferenceRules(roleBinding.RoleRef, namespace)
				if err != nil {
					if !visitor(nil, nil, err) {
						return
					}
					continue
				}
				sourceDescriber.binding = roleBinding
				sourceDescriber.subject = &roleBinding.Subjects[subjectIndex]
				for i := range rules {
					if !visitor(sourceDescriber, &rules[i], nil) {
						return
					}
				}
			}
		}
	}
}

// GetRoleReferenceRules attempts to resolve the RoleBinding or ClusterRoleBinding.
func (r *DefaultRuleResolver) GetRoleReferenceRules(roleRef rbac.RoleRef, bindingNamespace string) ([]rbac.PolicyRule, error) {
	switch roleRef.Kind {
	case "Role":
		role, err := r.roleGetter.GetRole(bindingNamespace, roleRef.Name)
		if err != nil {
			return nil, err
		}
		return role.Rules, nil

	case "ClusterRole":
		clusterRole, err := r.clusterRoleGetter.GetClusterRole(roleRef.Name)
		if err != nil {
			return nil, err
		}
		return clusterRole.Rules, nil

	default:
		return nil, fmt.Errorf("unsupported role reference kind: %q", roleRef.Kind)
	}
}

// appliesTo returns whether any of the bindingSubjects applies to the specified subject,
// and if true, the index of the first subject that applies
func appliesTo(user user.Info, bindingSubjects []rbac.Subject, namespace string) (int, bool) {
	for i, bindingSubject := range bindingSubjects {
		if appliesToUser(user, bindingSubject, namespace) {
			return i, true
		}
	}
	return 0, false
}

func has(set []string, ele string) bool {
	for _, s := range set {
		if s == ele {
			return true
		}
	}
	return false
}

func appliesToUser(user user.Info, subject rbac.Subject, namespace string) bool {
	switch subject.Kind {
	case rbac.UserKind:
		return user.GetName() == subject.Name

	case rbac.GroupKind:
		return has(user.GetGroups(), subject.Name)

	case rbac.ServiceAccountKind:
		// default the namespace to namespace we're working in if its available.  This allows rolebindings that reference
		// SAs in th local namespace to avoid having to qualify them.
		saNamespace := namespace
		if len(subject.Namespace) > 0 {
			saNamespace = subject.Namespace
		}
		if len(saNamespace) == 0 {
			return false
		}
		// use a more efficient comparison for RBAC checking
		return serviceaccount.MatchesUsername(saNamespace, subject.Name, user.GetName())
	default:
		return false
	}
}

// NewTestRuleResolver returns a rule resolver from lists of role objects.
func NewTestRuleResolver(roles []*rbac.Role, roleBindings []*rbac.RoleBinding, clusterRoles []*rbac.ClusterRole, clusterRoleBindings []*rbac.ClusterRoleBinding) (AuthorizationRuleResolver, *StaticRoles) {
	r := StaticRoles{
		roles:               roles,
		roleBindings:        roleBindings,
		clusterRoles:        clusterRoles,
		clusterRoleBindings: clusterRoleBindings,
	}
	return newMockRuleResolver(&r), &r
}

func newMockRuleResolver(r *StaticRoles) AuthorizationRuleResolver {
	return NewDefaultRuleResolver(r, r, r, r)
}

// StaticRoles is a rule resolver that resolves from lists of role objects.
type StaticRoles struct {
	roles               []*rbac.Role
	roleBindings        []*rbac.RoleBinding
	clusterRoles        []*rbac.ClusterRole
	clusterRoleBindings []*rbac.ClusterRoleBinding
}

func (r *StaticRoles) GetRole(namespace, name string) (*rbac.Role, error) {
	if len(namespace) == 0 {
		return nil, errors.New("must provide namespace when getting role")
	}
	for _, role := range r.roles {
		if role.Namespace == namespace && role.Name == name {
			return role, nil
		}
	}
	return nil, errors.New("role not found")
}

func (r *StaticRoles) GetClusterRole(name string) (*rbac.ClusterRole, error) {
	for _, clusterRole := range r.clusterRoles {
		if clusterRole.Name == name {
			return clusterRole, nil
		}
	}
	return nil, errors.New("clusterrole not found")
}

func (r *StaticRoles) ListRoleBindings(namespace string) ([]*rbac.RoleBinding, error) {
	if len(namespace) == 0 {
		return nil, errors.New("must provide namespace when listing role bindings")
	}

	roleBindingList := []*rbac.RoleBinding{}
	for _, roleBinding := range r.roleBindings {
		if roleBinding.Namespace != namespace {
			continue
		}
		// TODO(ericchiang): need to implement label selectors?
		roleBindingList = append(roleBindingList, roleBinding)
	}
	return roleBindingList, nil
}

func (r *StaticRoles) ListClusterRoleBindings() ([]*rbac.ClusterRoleBinding, error) {
	return r.clusterRoleBindings, nil
}