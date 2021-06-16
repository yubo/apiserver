/*
Copyright 2018 The Kubernetes Authors.

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

package v1

import (
	"fmt"
	"strings"

	"github.com/yubo/apiserver/pkg/api/rbac"
)

func VerbMatches(rule *rbac.PolicyRule, requestedVerb string) bool {
	for _, ruleVerb := range rule.Verbs {
		if ruleVerb == rbac.VerbAll {
			return true
		}
		if ruleVerb == requestedVerb {
			return true
		}
	}

	return false
}

//func APIGroupMatches(rule *rbac.PolicyRule, requestedGroup string) bool {
//	for _, ruleGroup := range rule.APIGroups {
//		if ruleGroup == rbac.APIGroupAll {
//			return true
//		}
//		if ruleGroup == requestedGroup {
//			return true
//		}
//	}
//
//	return false
//}

func ResourceMatches(rule *rbac.PolicyRule, combinedRequestedResource, requestedSubresource string) bool {
	for _, ruleResource := range rule.Resources {
		// if everything is allowed, we match
		if ruleResource == rbac.ResourceAll {
			return true
		}
		// if we have an exact match, we match
		if ruleResource == combinedRequestedResource {
			return true
		}

		// We can also match a */subresource.
		// if there isn't a subresource, then continue
		if len(requestedSubresource) == 0 {
			continue
		}
		// if the rule isn't in the format */subresource, then we don't match, continue
		if len(ruleResource) == len(requestedSubresource)+2 &&
			strings.HasPrefix(ruleResource, "*/") &&
			strings.HasSuffix(ruleResource, requestedSubresource) {
			return true

		}
	}

	return false
}

func ResourceNameMatches(rule *rbac.PolicyRule, requestedName string) bool {
	if len(rule.ResourceNames) == 0 {
		return true
	}

	for _, ruleName := range rule.ResourceNames {
		if ruleName == requestedName {
			return true
		}
	}

	return false
}

func NonResourceURLMatches(rule *rbac.PolicyRule, requestedURL string) bool {
	for _, ruleURL := range rule.NonResourceURLs {
		if ruleURL == rbac.NonResourceAll {
			return true
		}
		if ruleURL == requestedURL {
			return true
		}
		if strings.HasSuffix(ruleURL, "*") && strings.HasPrefix(requestedURL, strings.TrimRight(ruleURL, "*")) {
			return true
		}
	}

	return false
}

// subjectsStrings returns users, groups, serviceaccounts, unknown for display purposes.
func SubjectsStrings(subjects []rbac.Subject) ([]string, []string, []string, []string) {
	users := []string{}
	groups := []string{}
	sas := []string{}
	others := []string{}

	for _, subject := range subjects {
		switch subject.Kind {
		case rbac.ServiceAccountKind:
			sas = append(sas, fmt.Sprintf("%s/%s", subject.Namespace, subject.Name))

		case rbac.UserKind:
			users = append(users, subject.Name)

		case rbac.GroupKind:
			groups = append(groups, subject.Name)

		default:
			others = append(others, fmt.Sprintf("%s/%s/%s", subject.Kind, subject.Namespace, subject.Name))
		}
	}

	return users, groups, sas, others
}

func String(r rbac.PolicyRule) string {
	return "PolicyRule" + CompactString(r)
}

// CompactString exposes a compact string representation for use in escalation error messages
func CompactString(r rbac.PolicyRule) string {
	formatStringParts := []string{}
	formatArgs := []interface{}{}
	//if len(r.APIGroups) > 0 {
	//	formatStringParts = append(formatStringParts, "APIGroups:%q")
	//	formatArgs = append(formatArgs, r.APIGroups)
	//}
	if len(r.Resources) > 0 {
		formatStringParts = append(formatStringParts, "Resources:%q")
		formatArgs = append(formatArgs, r.Resources)
	}
	if len(r.NonResourceURLs) > 0 {
		formatStringParts = append(formatStringParts, "NonResourceURLs:%q")
		formatArgs = append(formatArgs, r.NonResourceURLs)
	}
	if len(r.ResourceNames) > 0 {
		formatStringParts = append(formatStringParts, "ResourceNames:%q")
		formatArgs = append(formatArgs, r.ResourceNames)
	}
	if len(r.Verbs) > 0 {
		formatStringParts = append(formatStringParts, "Verbs:%q")
		formatArgs = append(formatArgs, r.Verbs)
	}
	formatString := "{" + strings.Join(formatStringParts, ", ") + "}"
	return fmt.Sprintf(formatString, formatArgs...)
}

type SortableRuleSlice []rbac.PolicyRule

func (s SortableRuleSlice) Len() int      { return len(s) }
func (s SortableRuleSlice) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s SortableRuleSlice) Less(i, j int) bool {
	return strings.Compare(s[i].String(), s[j].String()) < 0
}
