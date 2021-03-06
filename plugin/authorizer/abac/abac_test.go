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

package abac

import (
	"context"
	"io/ioutil"
	"os"
	"reflect"
	"testing"

	"github.com/yubo/apiserver/pkg/authentication/user"
	"github.com/yubo/apiserver/plugin/authorizer/abac/api"
	"github.com/yubo/apiserver/pkg/authorization/authorizer"
	"github.com/yubo/golib/runtime"
)

func TestEmptyFile(t *testing.T) {
	_, err := newWithContents(t, "")
	if err != nil {
		t.Errorf("unable to read policy file: %v", err)
	}
}

func TestOneLineFileNoNewLine(t *testing.T) {
	_, err := newWithContents(t, `{"user":"scheduler",  "readonly": true, "resource": "pods", "namespace":"ns1"}`)
	if err != nil {
		t.Errorf("unable to read policy file: %v", err)
	}
}

func TestTwoLineFile(t *testing.T) {
	_, err := newWithContents(t, `{"user":"scheduler",  "readonly": true, "resource": "pods"}
{"user":"scheduler",  "readonly": true, "resource": "services"}
`)
	if err != nil {
		t.Errorf("unable to read policy file: %v", err)
	}
}

// Test the file that we will point users at as an example.
func TestExampleFile(t *testing.T) {
	_, err := NewFromFile("./example_policy_file.jsonl")
	if err != nil {
		t.Errorf("unable to read policy file: %v", err)
	}
}

func TestAuthorizeV0(t *testing.T) {
	a, err := newWithContents(t, `
{                    "readonly": true, "resource": "events"   }
{"user":"scheduler", "readonly": true, "resource": "pods"     }
{"user":"scheduler",                   "resource": "bindings" }
{"user":"kubelet",   "readonly": true, "resource": "bindings" }
{"user":"kubelet",                     "resource": "events"   }
{"user":"alice",                                              "namespace": "projectCaribou"}
{"user":"bob",       "readonly": true,                        "namespace": "projectCaribou"}
`)
	if err != nil {
		t.Fatalf("unable to read policy file: %v", err)
	}

	authenticatedGroup := []string{user.AllAuthenticated}

	uScheduler := user.DefaultInfo{Name: "scheduler", UID: "uid1", Groups: authenticatedGroup}
	uAlice := user.DefaultInfo{Name: "alice", UID: "uid3", Groups: authenticatedGroup}
	uChuck := user.DefaultInfo{Name: "chuck", UID: "uid5", Groups: authenticatedGroup}

	testCases := []struct {
		User           user.DefaultInfo
		Verb           string
		Resource       string
		NS             string
		APIGroup       string
		Path           string
		ExpectDecision authorizer.Decision
	}{
		// Scheduler can read pods
		{User: uScheduler, Verb: "list", Resource: "pods", NS: "ns1", ExpectDecision: authorizer.DecisionAllow},
		{User: uScheduler, Verb: "list", Resource: "pods", NS: "", ExpectDecision: authorizer.DecisionAllow},
		// Scheduler cannot write pods
		{User: uScheduler, Verb: "create", Resource: "pods", NS: "ns1", ExpectDecision: authorizer.DecisionNoOpinion},
		{User: uScheduler, Verb: "create", Resource: "pods", NS: "", ExpectDecision: authorizer.DecisionNoOpinion},
		// Scheduler can write bindings
		{User: uScheduler, Verb: "get", Resource: "bindings", NS: "ns1", ExpectDecision: authorizer.DecisionAllow},
		{User: uScheduler, Verb: "get", Resource: "bindings", NS: "", ExpectDecision: authorizer.DecisionAllow},

		// Alice can read and write anything in the right namespace.
		{User: uAlice, Verb: "get", Resource: "pods", NS: "projectCaribou", ExpectDecision: authorizer.DecisionAllow},
		{User: uAlice, Verb: "get", Resource: "widgets", NS: "projectCaribou", ExpectDecision: authorizer.DecisionAllow},
		{User: uAlice, Verb: "get", Resource: "", NS: "projectCaribou", ExpectDecision: authorizer.DecisionAllow},
		{User: uAlice, Verb: "update", Resource: "pods", NS: "projectCaribou", ExpectDecision: authorizer.DecisionAllow},
		{User: uAlice, Verb: "update", Resource: "widgets", NS: "projectCaribou", ExpectDecision: authorizer.DecisionAllow},
		{User: uAlice, Verb: "update", Resource: "", NS: "projectCaribou", ExpectDecision: authorizer.DecisionAllow},
		{User: uAlice, Verb: "update", Resource: "foo", NS: "projectCaribou", APIGroup: "bar", ExpectDecision: authorizer.DecisionAllow},
		// .. but not the wrong namespace.
		{User: uAlice, Verb: "get", Resource: "pods", NS: "ns1", ExpectDecision: authorizer.DecisionNoOpinion},
		{User: uAlice, Verb: "get", Resource: "widgets", NS: "ns1", ExpectDecision: authorizer.DecisionNoOpinion},
		{User: uAlice, Verb: "get", Resource: "", NS: "ns1", ExpectDecision: authorizer.DecisionNoOpinion},

		// Chuck can read events, since anyone can.
		{User: uChuck, Verb: "get", Resource: "events", NS: "ns1", ExpectDecision: authorizer.DecisionAllow},
		{User: uChuck, Verb: "get", Resource: "events", NS: "", ExpectDecision: authorizer.DecisionAllow},
		// Chuck can't do other things.
		{User: uChuck, Verb: "update", Resource: "events", NS: "ns1", ExpectDecision: authorizer.DecisionNoOpinion},
		{User: uChuck, Verb: "get", Resource: "pods", NS: "ns1", ExpectDecision: authorizer.DecisionNoOpinion},
		{User: uChuck, Verb: "get", Resource: "floop", NS: "ns1", ExpectDecision: authorizer.DecisionNoOpinion},
		// Chunk can't access things with no kind or namespace
		{User: uChuck, Verb: "get", Path: "/", Resource: "", NS: "", ExpectDecision: authorizer.DecisionNoOpinion},
	}
	for i, tc := range testCases {
		attr := authorizer.AttributesRecord{
			User:      &tc.User,
			Verb:      tc.Verb,
			Resource:  tc.Resource,
			Namespace: tc.NS,
			APIGroup:  tc.APIGroup,
			Path:      tc.Path,

			ResourceRequest: len(tc.NS) > 0 || len(tc.Resource) > 0,
		}
		decision, _, _ := a.Authorize(context.Background(), attr)
		if tc.ExpectDecision != decision {
			t.Logf("tc: %v -> attr %v", tc, attr)
			t.Errorf("%d: Expected allowed=%v but actually allowed=%v\n\t%v",
				i, tc.ExpectDecision, decision, tc)
		}
	}
}

func getResourceRules(infos []authorizer.ResourceRuleInfo) []authorizer.DefaultResourceRuleInfo {
	rules := make([]authorizer.DefaultResourceRuleInfo, len(infos))
	for i, info := range infos {
		rules[i] = authorizer.DefaultResourceRuleInfo{
			Verbs:         info.GetVerbs(),
			APIGroups:     info.GetAPIGroups(),
			Resources:     info.GetResources(),
			ResourceNames: info.GetResourceNames(),
		}
	}
	return rules
}

func getNonResourceRules(infos []authorizer.NonResourceRuleInfo) []authorizer.DefaultNonResourceRuleInfo {
	rules := make([]authorizer.DefaultNonResourceRuleInfo, len(infos))
	for i, info := range infos {
		rules[i] = authorizer.DefaultNonResourceRuleInfo{
			Verbs:           info.GetVerbs(),
			NonResourceURLs: info.GetNonResourceURLs(),
		}
	}
	return rules
}

func TestRulesFor(t *testing.T) {
	a, err := newWithContents(t, `
{                    "readonly": true, "resource": "events"   }
{"user":"scheduler", "readonly": true, "resource": "pods"     }
{"user":"scheduler",                   "resource": "bindings" }
{"user":"kubelet",   "readonly": true, "resource": "pods"     }
{"user":"kubelet",                     "resource": "events"   }
{"user":"alice",                                              "namespace": "projectCaribou"}
{"user":"bob",       "readonly": true,                        "namespace": "projectCaribou"}
{"user":"bob",       "readonly": true,                                                     "nonResourcePath": "*"}
{"group":"a",                          "resource": "bindings" }
{"group":"b",        "readonly": true,                                                     "nonResourcePath": "*"}
`)
	if err != nil {
		t.Fatalf("unable to read policy file: %v", err)
	}

	authenticatedGroup := []string{user.AllAuthenticated}

	uScheduler := user.DefaultInfo{Name: "scheduler", UID: "uid1", Groups: authenticatedGroup}
	uKubelet := user.DefaultInfo{Name: "kubelet", UID: "uid2", Groups: []string{"a", "b"}}
	uAlice := user.DefaultInfo{Name: "alice", UID: "uid3", Groups: authenticatedGroup}
	uBob := user.DefaultInfo{Name: "bob", UID: "uid4", Groups: authenticatedGroup}
	uChuck := user.DefaultInfo{Name: "chuck", UID: "uid5", Groups: []string{"a", "b"}}

	testCases := []struct {
		User                   user.DefaultInfo
		Namespace              string
		ExpectResourceRules    []authorizer.DefaultResourceRuleInfo
		ExpectNonResourceRules []authorizer.DefaultNonResourceRuleInfo
	}{
		{
			User:      uScheduler,
			Namespace: "ns1",
			ExpectResourceRules: []authorizer.DefaultResourceRuleInfo{
				{
					Verbs:     []string{"get", "list", "watch"},
					APIGroups: []string{"*"},
					Resources: []string{"events"},
				},
				{
					Verbs:     []string{"get", "list", "watch"},
					APIGroups: []string{"*"},
					Resources: []string{"pods"},
				},
				{
					Verbs:     []string{"*"},
					APIGroups: []string{"*"},
					Resources: []string{"bindings"},
				},
			},
			ExpectNonResourceRules: []authorizer.DefaultNonResourceRuleInfo{},
		},
		{
			User:      uKubelet,
			Namespace: "ns1",
			ExpectResourceRules: []authorizer.DefaultResourceRuleInfo{
				{
					Verbs:     []string{"get", "list", "watch"},
					APIGroups: []string{"*"},
					Resources: []string{"pods"},
				},
				{
					Verbs:     []string{"*"},
					APIGroups: []string{"*"},
					Resources: []string{"events"},
				},
				{
					Verbs:     []string{"*"},
					APIGroups: []string{"*"},
					Resources: []string{"bindings"},
				},
				{
					Verbs:     []string{"get", "list", "watch"},
					APIGroups: []string{"*"},
					Resources: []string{"*"},
				},
			},
			ExpectNonResourceRules: []authorizer.DefaultNonResourceRuleInfo{
				{
					Verbs:           []string{"get", "list", "watch"},
					NonResourceURLs: []string{"*"},
				},
			},
		},
		{
			User:      uAlice,
			Namespace: "projectCaribou",
			ExpectResourceRules: []authorizer.DefaultResourceRuleInfo{
				{
					Verbs:     []string{"get", "list", "watch"},
					APIGroups: []string{"*"},
					Resources: []string{"events"},
				},
				{
					Verbs:     []string{"*"},
					APIGroups: []string{"*"},
					Resources: []string{"*"},
				},
			},
			ExpectNonResourceRules: []authorizer.DefaultNonResourceRuleInfo{},
		},
		{
			User:      uBob,
			Namespace: "projectCaribou",
			ExpectResourceRules: []authorizer.DefaultResourceRuleInfo{
				{
					Verbs:     []string{"get", "list", "watch"},
					APIGroups: []string{"*"},
					Resources: []string{"events"},
				},
				{
					Verbs:     []string{"get", "list", "watch"},
					APIGroups: []string{"*"},
					Resources: []string{"*"},
				},
				{
					Verbs:     []string{"get", "list", "watch"},
					APIGroups: []string{"*"},
					Resources: []string{"*"},
				},
			},
			ExpectNonResourceRules: []authorizer.DefaultNonResourceRuleInfo{
				{
					Verbs:           []string{"get", "list", "watch"},
					NonResourceURLs: []string{"*"},
				},
			},
		},
		{
			User:      uChuck,
			Namespace: "ns1",
			ExpectResourceRules: []authorizer.DefaultResourceRuleInfo{
				{
					Verbs:     []string{"*"},
					APIGroups: []string{"*"},
					Resources: []string{"bindings"},
				},
				{
					Verbs:     []string{"get", "list", "watch"},
					APIGroups: []string{"*"},
					Resources: []string{"*"},
				},
			},
			ExpectNonResourceRules: []authorizer.DefaultNonResourceRuleInfo{
				{
					Verbs:           []string{"get", "list", "watch"},
					NonResourceURLs: []string{"*"},
				},
			},
		},
	}
	for i, tc := range testCases {
		attr := authorizer.AttributesRecord{
			User:      &tc.User,
			Namespace: tc.Namespace,
		}
		resourceRules, nonResourceRules, _, _ := a.RulesFor(attr.GetUser(), attr.GetNamespace())
		actualResourceRules := getResourceRules(resourceRules)
		if !reflect.DeepEqual(tc.ExpectResourceRules, actualResourceRules) {
			t.Logf("tc: %v -> attr %v", tc, attr)
			t.Errorf("%d: Expected: \n%#v\n but actual: \n%#v\n",
				i, tc.ExpectResourceRules, actualResourceRules)
		}
		actualNonResourceRules := getNonResourceRules(nonResourceRules)
		if !reflect.DeepEqual(tc.ExpectNonResourceRules, actualNonResourceRules) {
			t.Logf("tc: %v -> attr %v", tc, attr)
			t.Errorf("%d: Expected: \n%#v\n but actual: \n%#v\n",
				i, tc.ExpectNonResourceRules, actualNonResourceRules)
		}
	}
}

func TestAuthorizeV1beta1(t *testing.T) {
	a, err := newWithContents(t,
		`
		 # Comment line, after a blank line
		 {"apiVersion":"abac.authorization.kubernetes.io/v1beta1","kind":"Policy","spec":{"user":"*",         "readonly": true,                                                        "nonResourcePath": "/api"}}
		 {"apiVersion":"abac.authorization.kubernetes.io/v1beta1","kind":"Policy","spec":{"user":"*",                                                                                  "nonResourcePath": "/custom"}}
		 {"apiVersion":"abac.authorization.kubernetes.io/v1beta1","kind":"Policy","spec":{"user":"*",                                                                                  "nonResourcePath": "/root/*"}}
		 {"apiVersion":"abac.authorization.kubernetes.io/v1beta1","kind":"Policy","spec":{"user":"noresource",                                                                         "nonResourcePath": "*"}}
		 {"apiVersion":"abac.authorization.kubernetes.io/v1beta1","kind":"Policy","spec":{"user":"*",         "readonly": true, "resource": "events",   "namespace": "*"}}
		 {"apiVersion":"abac.authorization.kubernetes.io/v1beta1","kind":"Policy","spec":{"user":"scheduler", "readonly": true, "resource": "pods",     "namespace": "*"}}
		 {"apiVersion":"abac.authorization.kubernetes.io/v1beta1","kind":"Policy","spec":{"user":"scheduler",                   "resource": "bindings", "namespace": "*"}}
		 {"apiVersion":"abac.authorization.kubernetes.io/v1beta1","kind":"Policy","spec":{"user":"kubelet",   "readonly": true, "resource": "bindings", "namespace": "*"}}
		 {"apiVersion":"abac.authorization.kubernetes.io/v1beta1","kind":"Policy","spec":{"user":"kubelet",                     "resource": "events",   "namespace": "*"}}
		 {"apiVersion":"abac.authorization.kubernetes.io/v1beta1","kind":"Policy","spec":{"user":"alice",                       "resource": "*",        "namespace": "projectCaribou"}}
		 {"apiVersion":"abac.authorization.kubernetes.io/v1beta1","kind":"Policy","spec":{"user":"bob",       "readonly": true, "resource": "*",        "namespace": "projectCaribou"}}
		 {"apiVersion":"abac.authorization.kubernetes.io/v1beta1","kind":"Policy","spec":{"user":"debbie",                      "resource": "pods",     "namespace": "projectCaribou"}}
		 {"apiVersion":"abac.authorization.kubernetes.io/v1beta1","kind":"Policy","spec":{"user":"apigroupuser",                "resource": "*",        "namespace": "projectAnyGroup",   "apiGroup": "*"}}
		 {"apiVersion":"abac.authorization.kubernetes.io/v1beta1","kind":"Policy","spec":{"user":"apigroupuser",                "resource": "*",        "namespace": "projectEmptyGroup", "apiGroup": "" }}
		 {"apiVersion":"abac.authorization.kubernetes.io/v1beta1","kind":"Policy","spec":{"user":"apigroupuser",                "resource": "*",        "namespace": "projectXGroup",     "apiGroup": "x"}}`)

	if err != nil {
		t.Fatalf("unable to read policy file: %v", err)
	}

	authenticatedGroup := []string{user.AllAuthenticated}

	uScheduler := user.DefaultInfo{Name: "scheduler", UID: "uid1", Groups: authenticatedGroup}
	uAlice := user.DefaultInfo{Name: "alice", UID: "uid3", Groups: authenticatedGroup}
	uChuck := user.DefaultInfo{Name: "chuck", UID: "uid5", Groups: authenticatedGroup}
	uDebbie := user.DefaultInfo{Name: "debbie", UID: "uid6", Groups: authenticatedGroup}
	uNoResource := user.DefaultInfo{Name: "noresource", UID: "uid7", Groups: authenticatedGroup}
	uAPIGroup := user.DefaultInfo{Name: "apigroupuser", UID: "uid8", Groups: authenticatedGroup}

	testCases := []struct {
		User           user.DefaultInfo
		Verb           string
		Resource       string
		APIGroup       string
		NS             string
		Path           string
		ExpectDecision authorizer.Decision
	}{
		// Scheduler can read pods
		{User: uScheduler, Verb: "list", Resource: "pods", NS: "ns1", ExpectDecision: authorizer.DecisionAllow},
		{User: uScheduler, Verb: "list", Resource: "pods", NS: "", ExpectDecision: authorizer.DecisionAllow},
		// Scheduler cannot write pods
		{User: uScheduler, Verb: "create", Resource: "pods", NS: "ns1", ExpectDecision: authorizer.DecisionNoOpinion},
		{User: uScheduler, Verb: "create", Resource: "pods", NS: "", ExpectDecision: authorizer.DecisionNoOpinion},
		// Scheduler can write bindings
		{User: uScheduler, Verb: "get", Resource: "bindings", NS: "ns1", ExpectDecision: authorizer.DecisionAllow},
		{User: uScheduler, Verb: "get", Resource: "bindings", NS: "", ExpectDecision: authorizer.DecisionAllow},

		// Alice can read and write anything in the right namespace.
		{User: uAlice, Verb: "get", Resource: "pods", NS: "projectCaribou", ExpectDecision: authorizer.DecisionAllow},
		{User: uAlice, Verb: "get", Resource: "widgets", NS: "projectCaribou", ExpectDecision: authorizer.DecisionAllow},
		{User: uAlice, Verb: "get", Resource: "", NS: "projectCaribou", ExpectDecision: authorizer.DecisionAllow},
		{User: uAlice, Verb: "update", Resource: "pods", NS: "projectCaribou", ExpectDecision: authorizer.DecisionAllow},
		{User: uAlice, Verb: "update", Resource: "widgets", NS: "projectCaribou", ExpectDecision: authorizer.DecisionAllow},
		{User: uAlice, Verb: "update", Resource: "", NS: "projectCaribou", ExpectDecision: authorizer.DecisionAllow},
		// .. but not the wrong namespace.
		{User: uAlice, Verb: "get", Resource: "pods", NS: "ns1", ExpectDecision: authorizer.DecisionNoOpinion},
		{User: uAlice, Verb: "get", Resource: "widgets", NS: "ns1", ExpectDecision: authorizer.DecisionNoOpinion},
		{User: uAlice, Verb: "get", Resource: "", NS: "ns1", ExpectDecision: authorizer.DecisionNoOpinion},

		// Debbie can write to pods in the right namespace
		{User: uDebbie, Verb: "update", Resource: "pods", NS: "projectCaribou", ExpectDecision: authorizer.DecisionAllow},

		// Chuck can read events, since anyone can.
		{User: uChuck, Verb: "get", Resource: "events", NS: "ns1", ExpectDecision: authorizer.DecisionAllow},
		{User: uChuck, Verb: "get", Resource: "events", NS: "", ExpectDecision: authorizer.DecisionAllow},
		// Chuck can't do other things.
		{User: uChuck, Verb: "update", Resource: "events", NS: "ns1", ExpectDecision: authorizer.DecisionNoOpinion},
		{User: uChuck, Verb: "get", Resource: "pods", NS: "ns1", ExpectDecision: authorizer.DecisionNoOpinion},
		{User: uChuck, Verb: "get", Resource: "floop", NS: "ns1", ExpectDecision: authorizer.DecisionNoOpinion},
		// Chuck can't access things with no resource or namespace
		{User: uChuck, Verb: "get", Path: "/", Resource: "", NS: "", ExpectDecision: authorizer.DecisionNoOpinion},
		// but can access /api
		{User: uChuck, Verb: "get", Path: "/api", Resource: "", NS: "", ExpectDecision: authorizer.DecisionAllow},
		// though he cannot write to it
		{User: uChuck, Verb: "create", Path: "/api", Resource: "", NS: "", ExpectDecision: authorizer.DecisionNoOpinion},
		// while he can write to /custom
		{User: uChuck, Verb: "update", Path: "/custom", Resource: "", NS: "", ExpectDecision: authorizer.DecisionAllow},
		// he cannot get "/root"
		{User: uChuck, Verb: "get", Path: "/root", Resource: "", NS: "", ExpectDecision: authorizer.DecisionNoOpinion},
		// but can get any subpath
		{User: uChuck, Verb: "get", Path: "/root/", Resource: "", NS: "", ExpectDecision: authorizer.DecisionAllow},
		{User: uChuck, Verb: "get", Path: "/root/test/1/2/3", Resource: "", NS: "", ExpectDecision: authorizer.DecisionAllow},

		// the user "noresource" can get any non-resource request
		{User: uNoResource, Verb: "get", Path: "", Resource: "", NS: "", ExpectDecision: authorizer.DecisionAllow},
		{User: uNoResource, Verb: "get", Path: "/", Resource: "", NS: "", ExpectDecision: authorizer.DecisionAllow},
		{User: uNoResource, Verb: "get", Path: "/foo/bar/baz", Resource: "", NS: "", ExpectDecision: authorizer.DecisionAllow},
		// but cannot get any request where IsResourceRequest() == true
		{User: uNoResource, Verb: "get", Path: "/", Resource: "", NS: "bar", ExpectDecision: authorizer.DecisionNoOpinion},
		{User: uNoResource, Verb: "get", Path: "/foo/bar/baz", Resource: "foo", NS: "bar", ExpectDecision: authorizer.DecisionNoOpinion},

		// Test APIGroup matching
		{User: uAPIGroup, Verb: "get", APIGroup: "x", Resource: "foo", NS: "projectAnyGroup", ExpectDecision: authorizer.DecisionAllow},
		{User: uAPIGroup, Verb: "get", APIGroup: "x", Resource: "foo", NS: "projectEmptyGroup", ExpectDecision: authorizer.DecisionNoOpinion},
		{User: uAPIGroup, Verb: "get", APIGroup: "x", Resource: "foo", NS: "projectXGroup", ExpectDecision: authorizer.DecisionAllow},
	}
	for i, tc := range testCases {
		attr := authorizer.AttributesRecord{
			User:            &tc.User,
			Verb:            tc.Verb,
			Resource:        tc.Resource,
			APIGroup:        tc.APIGroup,
			Namespace:       tc.NS,
			ResourceRequest: len(tc.NS) > 0 || len(tc.Resource) > 0,
			Path:            tc.Path,
		}
		// t.Logf("tc %2v: %v -> attr %v", i, tc, attr)
		decision, _, _ := a.Authorize(context.Background(), attr)
		if tc.ExpectDecision != decision {
			t.Errorf("%d: Expected allowed=%v but actually allowed=%v, for case %+v & %+v",
				i, tc.ExpectDecision, decision, tc, attr)
		}
	}
}

func newWithContents(t *testing.T, contents string) (PolicyList, error) {
	f, err := ioutil.TempFile("", "abac_test")
	if err != nil {
		t.Fatalf("unexpected error creating policyfile: %v", err)
	}
	f.Close()
	defer os.Remove(f.Name())

	if err := ioutil.WriteFile(f.Name(), []byte(contents), 0700); err != nil {
		t.Fatalf("unexpected error writing policyfile: %v", err)
	}

	pl, err := NewFromFile(f.Name())
	return pl, err
}

func TestPolicy(t *testing.T) {
	tests := []struct {
		policy  runtime.Object
		attr    authorizer.Attributes
		matches bool
		name    string
	}{

		// v1 mismatches
		{
			policy: &api.Policy{},
			attr: authorizer.AttributesRecord{
				User: &user.DefaultInfo{
					Name:   "foo",
					Groups: []string{user.AllAuthenticated},
				},
				ResourceRequest: true,
			},
			matches: false,
			name:    "v1 null",
		},
		{
			policy: &api.Policy{
				Spec: api.PolicySpec{
					User: "foo",
				},
			},
			attr: authorizer.AttributesRecord{
				User: &user.DefaultInfo{
					Name:   "bar",
					Groups: []string{user.AllAuthenticated},
				},
				ResourceRequest: true,
			},
			matches: false,
			name:    "v1 user name mis-match",
		},
		{
			policy: &api.Policy{
				Spec: api.PolicySpec{
					User:     "*",
					Readonly: true,
				},
			},
			attr: authorizer.AttributesRecord{
				User: &user.DefaultInfo{
					Name:   "foo",
					Groups: []string{user.AllAuthenticated},
				},
				ResourceRequest: true,
			},
			matches: false,
			name:    "v1 read-only mismatch",
		},
		{
			policy: &api.Policy{
				Spec: api.PolicySpec{
					User:     "*",
					Resource: "foo",
				},
			},
			attr: authorizer.AttributesRecord{
				User: &user.DefaultInfo{
					Name:   "foo",
					Groups: []string{user.AllAuthenticated},
				},
				Resource:        "bar",
				ResourceRequest: true,
			},
			matches: false,
			name:    "v1 resource mis-match",
		},
		{
			policy: &api.Policy{
				Spec: api.PolicySpec{
					User:      "foo",
					Namespace: "barr",
					Resource:  "baz",
				},
			},
			attr: authorizer.AttributesRecord{
				User: &user.DefaultInfo{
					Name:   "foo",
					Groups: []string{user.AllAuthenticated},
				},
				Namespace:       "bar",
				Resource:        "baz",
				ResourceRequest: true,
			},
			matches: false,
			name:    "v1 namespace mis-match",
		},
		{
			policy: &api.Policy{
				Spec: api.PolicySpec{
					User:            "*",
					NonResourcePath: "/api",
				},
			},
			attr: authorizer.AttributesRecord{
				User: &user.DefaultInfo{
					Name:   "foo",
					Groups: []string{user.AllAuthenticated},
				},
				Path:            "/api2",
				ResourceRequest: false,
			},
			matches: false,
			name:    "v1 non-resource mis-match",
		},
		{
			policy: &api.Policy{
				Spec: api.PolicySpec{
					User:            "*",
					NonResourcePath: "/api/*",
				},
			},
			attr: authorizer.AttributesRecord{
				User: &user.DefaultInfo{
					Name:   "foo",
					Groups: []string{user.AllAuthenticated},
				},
				Path:            "/api2/foo",
				ResourceRequest: false,
			},
			matches: false,
			name:    "v1 non-resource wildcard subpath mis-match",
		},

		// v1 matches
		{
			policy: &api.Policy{
				Spec: api.PolicySpec{
					User: "foo",
				},
			},
			attr: authorizer.AttributesRecord{
				User: &user.DefaultInfo{
					Name:   "foo",
					Groups: []string{user.AllAuthenticated},
				},
				ResourceRequest: true,
			},
			matches: true,
			name:    "v1 user match",
		},
		{
			policy: &api.Policy{
				Spec: api.PolicySpec{
					User: "*",
				},
			},
			attr: authorizer.AttributesRecord{
				User: &user.DefaultInfo{
					Name:   "foo",
					Groups: []string{user.AllAuthenticated},
				},
				ResourceRequest: true,
			},
			matches: true,
			name:    "v1 user wildcard match",
		},
		{
			policy: &api.Policy{
				Spec: api.PolicySpec{
					Group: "bar",
				},
			},
			attr: authorizer.AttributesRecord{
				User: &user.DefaultInfo{
					Name:   "foo",
					Groups: []string{"bar", user.AllAuthenticated},
				},
				ResourceRequest: true,
			},
			matches: true,
			name:    "v1 group match",
		},
		{
			policy: &api.Policy{
				Spec: api.PolicySpec{
					Group: "*",
				},
			},
			attr: authorizer.AttributesRecord{
				User: &user.DefaultInfo{
					Name:   "foo",
					Groups: []string{"bar", user.AllAuthenticated},
				},
				ResourceRequest: true,
			},
			matches: true,
			name:    "v1 group wildcard match",
		},
		{
			policy: &api.Policy{
				Spec: api.PolicySpec{
					User:     "*",
					Readonly: true,
				},
			},
			attr: authorizer.AttributesRecord{
				User: &user.DefaultInfo{
					Name:   "foo",
					Groups: []string{user.AllAuthenticated},
				},
				Verb:            "get",
				ResourceRequest: true,
			},
			matches: true,
			name:    "v1 read-only match",
		},
		{
			policy: &api.Policy{
				Spec: api.PolicySpec{
					User:     "*",
					Resource: "foo",
				},
			},
			attr: authorizer.AttributesRecord{
				User: &user.DefaultInfo{
					Name:   "foo",
					Groups: []string{user.AllAuthenticated},
				},
				Resource:        "foo",
				ResourceRequest: true,
			},
			matches: true,
			name:    "v1 resource match",
		},
		{
			policy: &api.Policy{
				Spec: api.PolicySpec{
					User:      "foo",
					Namespace: "bar",
					Resource:  "baz",
				},
			},
			attr: authorizer.AttributesRecord{
				User: &user.DefaultInfo{
					Name:   "foo",
					Groups: []string{user.AllAuthenticated},
				},
				Namespace:       "bar",
				Resource:        "baz",
				ResourceRequest: true,
			},
			matches: true,
			name:    "v1 namespace match",
		},
		{
			policy: &api.Policy{
				Spec: api.PolicySpec{
					User:            "*",
					NonResourcePath: "/api",
				},
			},
			attr: authorizer.AttributesRecord{
				User: &user.DefaultInfo{
					Name:   "foo",
					Groups: []string{user.AllAuthenticated},
				},
				Path:            "/api",
				ResourceRequest: false,
			},
			matches: true,
			name:    "v1 non-resource match",
		},
		{
			policy: &api.Policy{
				Spec: api.PolicySpec{
					User:            "*",
					NonResourcePath: "*",
				},
			},
			attr: authorizer.AttributesRecord{
				User: &user.DefaultInfo{
					Name:   "foo",
					Groups: []string{user.AllAuthenticated},
				},
				Path:            "/api",
				ResourceRequest: false,
			},
			matches: true,
			name:    "v1 non-resource wildcard match",
		},
		{
			policy: &api.Policy{
				Spec: api.PolicySpec{
					User:            "*",
					NonResourcePath: "/api/*",
				},
			},
			attr: authorizer.AttributesRecord{
				User: &user.DefaultInfo{
					Name:   "foo",
					Groups: []string{user.AllAuthenticated},
				},
				Path:            "/api/foo",
				ResourceRequest: false,
			},
			matches: true,
			name:    "v1 non-resource wildcard subpath match",
		},
	}
	for _, test := range tests {
		policy := test.policy.(*api.Policy)
		matches := matches(*policy, test.attr)
		if test.matches != matches {
			t.Errorf("%s: expected: %t, saw: %t", test.name, test.matches, matches)
			continue
		}
	}
}
