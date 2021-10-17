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
	"fmt"
	"io/ioutil"

	auditinternal "github.com/yubo/apiserver/pkg/apis/audit"
	"github.com/yubo/apiserver/pkg/apis/audit/validation"
	"github.com/yubo/golib/scheme"

	"k8s.io/klog/v2"
)

func LoadPolicyFromFile(filePath string) (*auditinternal.Policy, error) {
	if filePath == "" {
		return nil, fmt.Errorf("file path not specified")
	}
	policyDef, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file path %q: %+v", filePath, err)
	}

	ret, err := LoadPolicyFromBytes(policyDef)
	if err != nil {
		return nil, fmt.Errorf("%v: from file %v", err.Error(), filePath)
	}

	return ret, nil
}

func LoadPolicyFromBytes(policyDef []byte) (*auditinternal.Policy, error) {
	policy := &auditinternal.Policy{}
	decoder := scheme.Codecs.UniversalDecoder()

	_, err := decoder.Decode(policyDef, policy)
	if err != nil {
		return nil, fmt.Errorf("failed decoding: %v", err)
	}

	// Ensure the policy file contained an apiVersion and kind.
	//if !apiGroupVersionSet[schema.GroupVersion{Group: gvk.Group, Version: gvk.Version}] {
	//	return nil, fmt.Errorf("unknown group version field %v in policy", gvk)
	//}

	if err := validation.ValidatePolicy(policy); err != nil {
		return nil, err.ToAggregate()
	}

	policyCnt := len(policy.Rules)
	if policyCnt == 0 {
		return nil, fmt.Errorf("loaded illegal policy with 0 rules")
	}

	klog.V(4).InfoS("Load audit policy rules success", "policyCnt", policyCnt)
	return policy, nil
}
