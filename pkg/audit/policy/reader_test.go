/*
Copyright 2017 The Kubernetes Authors.

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

package policy

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/yubo/apiserver/pkg/apis/audit"
	"github.com/yubo/golib/api"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const policyDefPattern = `
apiVersion: audit.k8s.io/{version}
kind: Policy
rules:
  - level: None
    nonResourceURLs:
      - /healthz*
      - /version
  - level: RequestResponse
    users: ["tim"]
    userGroups: ["testers", "developers"]
    verbs: ["patch", "delete", "create"]
    resources:
      - group: ""
      - group: "rbac.authorization.k8s.io"
        resources: ["clusterroles", "clusterrolebindings"]
    namespaces: ["default", "kube-system"]
  - level: Metadata
`

const policyWithNoVersionOrKind = `
rules:
  - level: None
    nonResourceURLs:
      - /healthz*
      - /version
  - level: RequestResponse
    users: ["tim"]
    userGroups: ["testers", "developers"]
    verbs: ["patch", "delete", "create"]
    resources:
      - group: ""
      - group: "rbac.authorization.k8s.io"
        resources: ["clusterroles", "clusterrolebindings"]
    namespaces: ["default", "kube-system"]
  - level: Metadata
`

var expectedPolicy = &audit.Policy{
	Rules: []audit.PolicyRule{{
		Level:           audit.LevelNone,
		NonResourceURLs: []string{"/healthz*", "/version"},
	}, {
		Level:      audit.LevelRequestResponse,
		Users:      []string{"tim"},
		UserGroups: []string{"testers", "developers"},
		Verbs:      []string{"patch", "delete", "create"},
		Resources: []audit.GroupResources{{}, {
			Group:     "rbac.authorization.k8s.io",
			Resources: []string{"clusterroles", "clusterrolebindings"},
		}},
		Namespaces: []string{"default", "kube-system"},
	}, {
		Level: audit.LevelMetadata,
	}},
}

func TestParser(t *testing.T) {
	for _, version := range []string{"v1", "v1alpha1", "v1beta1"} {
		policyDef := strings.Replace(policyDefPattern, "{version}", version, 1)
		f, err := writePolicy(t, policyDef)
		require.NoError(t, err)
		defer os.Remove(f)

		policy, err := LoadPolicyFromFile(f)
		require.NoError(t, err)

		// ugly hack
		policy.TypeMeta = api.TypeMeta{}

		assert.Len(t, policy.Rules, 3) // Sanity check.
		assert.Equal(t, policy, expectedPolicy)
	}
}

//func TestParsePolicyWithNoVersionOrKind(t *testing.T) {
//	f, err := writePolicy(t, policyWithNoVersionOrKind)
//	require.NoError(t, err)
//	defer os.Remove(f)
//
//	_, err = LoadPolicyFromFile(f)
//	assert.Contains(t, err.Error(), "unknown group version field")
//}

func TestPolicyCntCheck(t *testing.T) {
	var testCases = []struct {
		caseName, policy string
	}{
		{
			"policyWithNoRule",
			`apiVersion: audit.k8s.io/v1
kind: Policy`,
		},
		{"emptyPolicyFile", ""},
	}

	for _, tc := range testCases {
		f, err := writePolicy(t, tc.policy)
		require.NoError(t, err)
		defer os.Remove(f)

		_, err = LoadPolicyFromFile(f)
		assert.Errorf(t, err, "loaded illegal policy with 0 rules from testCase %s", tc.caseName)
	}
}

func writePolicy(t *testing.T, policy string) (string, error) {
	f, err := ioutil.TempFile("", "policy.yaml")
	require.NoError(t, err)

	_, err = f.WriteString(policy)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	return f.Name(), nil
}
