/*
Copyright 2015 The Kubernetes Authors.

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

package api

import "errors"

// +k8s:deepcopy-gen:interfaces=github.com/yubo/golib/runtime.Object

// Policy contains a single ABAC policy rule
type Policy struct {
	//metav1.TypeMeta
	// Kind is a string value representing the REST resource this object represents.
	Kind string `json:"kind,omitempty"`

	// Spec describes the policy rule
	Spec PolicySpec `json:"spec,omitempty"`
}

func (p *Policy) Validate() error {
	if p.Spec.User == "" && p.Spec.Group == "" {
		return errors.New("user and group both empty")
	}

	return nil
}

// PolicySpec contains the attributes for a policy rule
type PolicySpec struct {

	// User is the username this rule applies to.
	// Either user or group is required to match the request.
	// "*" matches all users.
	User string `json:"user,omitempty"`

	// Group is the group this rule applies to.
	// Either user or group is required to match the request.
	// "*" matches all groups.
	Group string `json:"group,omitempty"`

	// Readonly matches readonly requests when true, and all requests when false
	Readonly bool `json:"readonly,omitempty"`

	// APIGroup is the name of an API group. APIGroup, Resource, and Namespace are required to match resource requests.
	// "*" matches all API groups
	APIGroup string `json:"apiGroup,omitempty"`

	// Resource is the name of a resource. APIGroup, Resource, and Namespace are required to match resource requests.
	// "*" matches all resources
	Resource string `json:"resource,omitempty"`

	// Namespace is the name of a namespace. APIGroup, Resource, and Namespace are required to match resource requests.
	// "*" matches all namespaces (including unnamespaced requests)
	Namespace string `json:"namespace,omitempty"`

	// NonResourcePath matches non-resource request paths.
	// "*" matches all paths
	// "/foo/*" matches all subpaths of foo
	NonResourcePath string `json:"nonResourcePath,omitempty"`

	// TODO: "expires" string in RFC3339 format.

	// TODO: want a way to allow some users to restart containers of a pod but
	// not delete or modify it.

	// TODO: want a way to allow a controller to create a pod based only on a
	// certain podTemplates.

}

func (p *PolicySpec) ValidateV0() error {
	p.APIGroup = "*"

	if p.Namespace == "" {
		p.Namespace = "*"
	}
	if p.Resource == "" {
		p.Resource = "*"
	}
	if p.User == "" && p.Group == "" {
		p.Group = "system:authenticated"
	}

	return nil
}
