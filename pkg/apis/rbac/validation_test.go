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

package rbac

import (
	"testing"

	"github.com/yubo/golib/api"
	"github.com/yubo/golib/util/validation/field"
)

func TestValidateClusterRoleBinding(t *testing.T) {
	errs := ValidateClusterRoleBinding(
		&ClusterRoleBinding{
			ObjectMeta: api.ObjectMeta{Name: "master"},
			RoleRef:    RoleRef{Kind: "ClusterRole", Name: "valid"},
			Subjects: []Subject{
				{Name: "validsaname", Namespace: "foo", Kind: ServiceAccountKind},
				{Name: "valid@username", Kind: UserKind},
				{Name: "valid@groupname", Kind: GroupKind},
			},
		},
	)
	if len(errs) != 0 {
		t.Errorf("expected success: %v", errs)
	}

	errorCases := map[string]struct {
		A ClusterRoleBinding
		T field.ErrorType
		F string
	}{
		//"bad group": {
		//	A: ClusterRoleBinding{
		//		ObjectMeta: api.ObjectMeta{Name: "default"},
		//		RoleRef:    RoleRef{Kind: "ClusterRole", Name: "valid"},
		//	},
		//	T: field.ErrorTypeNotSupported,
		//	F: "roleRef.apiGroup",
		//},
		"bad kind": {
			A: ClusterRoleBinding{
				ObjectMeta: api.ObjectMeta{Name: "default"},
				RoleRef:    RoleRef{Kind: "Type", Name: "valid"},
			},
			T: field.ErrorTypeNotSupported,
			F: "roleRef.kind",
		},
		"reference role": {
			A: ClusterRoleBinding{
				ObjectMeta: api.ObjectMeta{Name: "default"},
				RoleRef:    RoleRef{Kind: "Role", Name: "valid"},
			},
			T: field.ErrorTypeNotSupported,
			F: "roleRef.kind",
		},
		//"zero-length name": {
		//	A: ClusterRoleBinding{
		//		ObjectMeta: api.ObjectMeta{},
		//		RoleRef:    RoleRef{Kind: "ClusterRole", Name: "valid"},
		//	},
		//	T: field.ErrorTypeRequired,
		//	F: "metadata.name",
		//},
		"bad role": {
			A: ClusterRoleBinding{
				ObjectMeta: api.ObjectMeta{Name: "default"},
				RoleRef:    RoleRef{Kind: "ClusterRole"},
			},
			T: field.ErrorTypeRequired,
			F: "roleRef.name",
		},
		"bad subject kind": {
			A: ClusterRoleBinding{
				ObjectMeta: api.ObjectMeta{Name: "master"},
				RoleRef:    RoleRef{Kind: "ClusterRole", Name: "valid"},
				Subjects:   []Subject{{Name: "subject"}},
			},
			T: field.ErrorTypeNotSupported,
			F: "subjects[0].kind",
		},
		"bad subject name": {
			A: ClusterRoleBinding{
				ObjectMeta: api.ObjectMeta{Name: "master"},
				RoleRef:    RoleRef{Kind: "ClusterRole", Name: "valid"},
				Subjects:   []Subject{{Namespace: "foo", Name: "subject:bad", Kind: ServiceAccountKind}},
			},
			T: field.ErrorTypeInvalid,
			F: "subjects[0].name",
		},
		"missing SA namespace": {
			A: ClusterRoleBinding{
				ObjectMeta: api.ObjectMeta{Name: "master"},
				RoleRef:    RoleRef{Kind: "ClusterRole", Name: "valid"},
				Subjects:   []Subject{{Name: "good", Kind: ServiceAccountKind}},
			},
			T: field.ErrorTypeRequired,
			F: "subjects[0].namespace",
		},
		"missing subject name": {
			A: ClusterRoleBinding{
				ObjectMeta: api.ObjectMeta{Name: "master"},
				RoleRef:    RoleRef{Kind: "ClusterRole", Name: "valid"},
				Subjects:   []Subject{{Namespace: "foo", Kind: ServiceAccountKind}},
			},
			T: field.ErrorTypeRequired,
			F: "subjects[0].name",
		},
	}
	for k, v := range errorCases {
		errs := ValidateClusterRoleBinding(&v.A)
		if len(errs) == 0 {
			t.Errorf("expected failure %s for %v", k, v.A)
			continue
		}
		for i := range errs {
			if errs[i].Type != v.T {
				t.Errorf("%s: expected errors to have type %s: %v", k, v.T, errs[i])
			}
			if errs[i].Field != v.F {
				t.Errorf("%s: expected errors to have field %s: %v", k, v.F, errs[i])
			}
		}
	}
}

func TestValidateRoleBinding(t *testing.T) {
	errs := ValidateRoleBinding(
		&RoleBinding{
			ObjectMeta: api.ObjectMeta{Namespace: api.NamespaceDefault, Name: "master"},
			RoleRef:    RoleRef{Kind: "Role", Name: "valid"},
			Subjects: []Subject{
				{Name: "validsaname", Kind: ServiceAccountKind},
				{Name: "valid@username", Kind: UserKind},
				{Name: "valid@groupname", Kind: GroupKind},
			},
		},
	)
	if len(errs) != 0 {
		t.Errorf("expected success: %v", errs)
	}

	errorCases := map[string]struct {
		A RoleBinding
		T field.ErrorType
		F string
	}{
		//"bad group": {
		//	A: RoleBinding{
		//		ObjectMeta: api.ObjectMeta{Namespace: api.NamespaceDefault, Name: "default"},
		//		RoleRef:    RoleRef{Kind: "ClusterRole", Name: "valid"},
		//	},
		//	T: field.ErrorTypeNotSupported,
		//	F: "roleRef.apiGroup",
		//},
		"bad kind": {
			A: RoleBinding{
				ObjectMeta: api.ObjectMeta{Namespace: api.NamespaceDefault, Name: "default"},
				RoleRef:    RoleRef{Kind: "Type", Name: "valid"},
			},
			T: field.ErrorTypeNotSupported,
			F: "roleRef.kind",
		},
		//"zero-length namespace": {
		//	A: RoleBinding{
		//		ObjectMeta: api.ObjectMeta{Name: "default"},
		//		RoleRef:    RoleRef{Kind: "Role", Name: "valid"},
		//	},
		//	T: field.ErrorTypeRequired,
		//	F: "metadata.namespace",
		//},
		//"zero-length name": {
		//	A: RoleBinding{
		//		ObjectMeta: api.ObjectMeta{Namespace: api.NamespaceDefault},
		//		RoleRef:    RoleRef{Kind: "Role", Name: "valid"},
		//	},
		//	T: field.ErrorTypeRequired,
		//	F: "metadata.name",
		//},
		"bad role": {
			A: RoleBinding{
				ObjectMeta: api.ObjectMeta{Namespace: api.NamespaceDefault, Name: "default"},
				RoleRef:    RoleRef{Kind: "Role"},
			},
			T: field.ErrorTypeRequired,
			F: "roleRef.name",
		},
		"bad subject kind": {
			A: RoleBinding{
				ObjectMeta: api.ObjectMeta{Namespace: api.NamespaceDefault, Name: "master"},
				RoleRef:    RoleRef{Kind: "Role", Name: "valid"},
				Subjects:   []Subject{{Name: "subject"}},
			},
			T: field.ErrorTypeNotSupported,
			F: "subjects[0].kind",
		},
		"bad subject name": {
			A: RoleBinding{
				ObjectMeta: api.ObjectMeta{Namespace: api.NamespaceDefault, Name: "master"},
				RoleRef:    RoleRef{Kind: "Role", Name: "valid"},
				Subjects:   []Subject{{Name: "subject:bad", Kind: ServiceAccountKind}},
			},
			T: field.ErrorTypeInvalid,
			F: "subjects[0].name",
		},
		"missing subject name": {
			A: RoleBinding{
				ObjectMeta: api.ObjectMeta{Namespace: api.NamespaceDefault, Name: "master"},
				RoleRef:    RoleRef{Kind: "Role", Name: "valid"},
				Subjects:   []Subject{{Kind: ServiceAccountKind}},
			},
			T: field.ErrorTypeRequired,
			F: "subjects[0].name",
		},
	}
	for k, v := range errorCases {
		errs := ValidateRoleBinding(&v.A)
		if len(errs) == 0 {
			t.Errorf("expected failure %s for %v", k, v.A)
			continue
		}
		for i := range errs {
			if errs[i].Type != v.T {
				t.Errorf("%s: expected errors to have type %s: %v", k, v.T, errs[i])
			}
			if errs[i].Field != v.F {
				t.Errorf("%s: expected errors to have field %s: %v", k, v.F, errs[i])
			}
		}
	}
}

func TestValidateRoleBindingUpdate(t *testing.T) {
	old := &RoleBinding{
		ObjectMeta: api.ObjectMeta{Namespace: api.NamespaceDefault, Name: "master", ResourceVersion: "1"},
		RoleRef:    RoleRef{Kind: "Role", Name: "valid"},
	}

	errs := ValidateRoleBindingUpdate(
		&RoleBinding{
			ObjectMeta: api.ObjectMeta{Namespace: api.NamespaceDefault, Name: "master", ResourceVersion: "1"},
			RoleRef:    RoleRef{Kind: "Role", Name: "valid"},
		},
		old,
	)
	if len(errs) != 0 {
		t.Errorf("expected success: %v", errs)
	}

	errorCases := map[string]struct {
		A RoleBinding
		T field.ErrorType
		F string
	}{
		"changedRef": {
			A: RoleBinding{
				ObjectMeta: api.ObjectMeta{Namespace: api.NamespaceDefault, Name: "master", ResourceVersion: "1"},
				RoleRef:    RoleRef{Kind: "Role", Name: "changed"},
			},
			T: field.ErrorTypeInvalid,
			F: "roleRef",
		},
	}
	for k, v := range errorCases {
		errs := ValidateRoleBindingUpdate(&v.A, old)
		if len(errs) == 0 {
			t.Errorf("expected failure %s for %v", k, v.A)
			continue
		}
		for i := range errs {
			if errs[i].Type != v.T {
				t.Errorf("%s: expected errors to have type %s: %v", k, v.T, errs[i])
			}
			if errs[i].Field != v.F {
				t.Errorf("%s: expected errors to have field %s: %v", k, v.F, errs[i])
			}
		}
	}
}

type ValidateRoleTest struct {
	role    Role
	wantErr bool
	errType field.ErrorType
	field   string
}

func (v ValidateRoleTest) test(t *testing.T) {
	errs := ValidateRole(&v.role)
	if len(errs) == 0 {
		if v.wantErr {
			t.Fatal("expected validation error")
		}
		return
	}
	if !v.wantErr {
		t.Errorf("didn't expect error, got %v", errs)
		return
	}
	for i := range errs {
		if errs[i].Type != v.errType {
			t.Errorf("expected errors to have type %s: %v", v.errType, errs[i])
		}
		if errs[i].Field != v.field {
			t.Errorf("expected errors to have field %s: %v", v.field, errs[i])
		}
	}
}

type ValidateClusterRoleTest struct {
	role    ClusterRole
	wantErr bool
	errType field.ErrorType
	field   string
}

func (v ValidateClusterRoleTest) test(t *testing.T) {
	errs := ValidateClusterRole(&v.role)
	if len(errs) == 0 {
		if v.wantErr {
			t.Fatal("expected validation error")
		}
		return
	}
	if !v.wantErr {
		t.Errorf("didn't expect error, got %v", errs)
		return
	}
	for i := range errs {
		if errs[i].Type != v.errType {
			t.Errorf("expected errors to have type %s: %v", v.errType, errs[i])
		}
		if errs[i].Field != v.field {
			t.Errorf("expected errors to have field %s: %v", v.field, errs[i])
		}
	}
}

//func TestValidateRoleZeroLengthNamespace(t *testing.T) {
//	ValidateRoleTest{
//		role: Role{
//			ObjectMeta: api.ObjectMeta{Name: "default"},
//		},
//		wantErr: true,
//		errType: field.ErrorTypeRequired,
//		field:   "metadata.namespace",
//	}.test(t)
//}

//func TestValidateRoleZeroLengthName(t *testing.T) {
//	ValidateRoleTest{
//		role: Role{
//			ObjectMeta: api.ObjectMeta{Namespace: "default"},
//		},
//		wantErr: true,
//		errType: field.ErrorTypeRequired,
//		field:   "metadata.name",
//	}.test(t)
//}

func TestValidateRoleValidRole(t *testing.T) {
	ValidateRoleTest{
		role: Role{
			ObjectMeta: api.ObjectMeta{
				Namespace: "default",
				Name:      "default",
			},
		},
		wantErr: false,
	}.test(t)
}

func TestValidateRoleValidRoleNoNamespace(t *testing.T) {
	ValidateClusterRoleTest{
		role: ClusterRole{
			ObjectMeta: api.ObjectMeta{
				Name: "default",
			},
		},
		wantErr: false,
	}.test(t)
}

func TestValidateRoleNonResourceURL(t *testing.T) {
	ValidateClusterRoleTest{
		role: ClusterRole{
			ObjectMeta: api.ObjectMeta{
				Name: "default",
			},
			Rules: []PolicyRule{
				{
					Verbs:           []string{"get"},
					NonResourceURLs: []string{"/*"},
				},
			},
		},
		wantErr: false,
	}.test(t)
}

func TestValidateRoleNamespacedNonResourceURL(t *testing.T) {
	ValidateRoleTest{
		role: Role{
			ObjectMeta: api.ObjectMeta{
				Namespace: "default",
				Name:      "default",
			},
			Rules: []PolicyRule{
				{
					// non-resource URLs are invalid for namespaced rules
					Verbs:           []string{"get"},
					NonResourceURLs: []string{"/*"},
				},
			},
		},
		wantErr: true,
		errType: field.ErrorTypeInvalid,
		field:   "rules[0].nonResourceURLs",
	}.test(t)
}

//func TestValidateRoleNonResourceURLNoVerbs(t *testing.T) {
//	ValidateClusterRoleTest{
//		role: ClusterRole{
//			ObjectMeta: api.ObjectMeta{
//				Name: "default",
//			},
//			Rules: []PolicyRule{
//				{
//					Verbs:           []string{},
//					NonResourceURLs: []string{"/*"},
//				},
//			},
//		},
//		wantErr: true,
//		errType: field.ErrorTypeRequired,
//		field:   "rules[0].verbs",
//	}.test(t)
//}

func TestValidateRoleMixedNonResourceAndResource(t *testing.T) {
	ValidateRoleTest{
		role: Role{
			ObjectMeta: api.ObjectMeta{
				Name:      "default",
				Namespace: "default",
			},
			Rules: []PolicyRule{
				{
					Verbs:           []string{"get"},
					NonResourceURLs: []string{"/*"},
					Resources:       []string{"pods"},
				},
			},
		},
		wantErr: true,
		errType: field.ErrorTypeInvalid,
		field:   "rules[0].nonResourceURLs",
	}.test(t)
}

func TestValidateRoleValidResource(t *testing.T) {
	ValidateRoleTest{
		role: Role{
			ObjectMeta: api.ObjectMeta{
				Name:      "default",
				Namespace: "default",
			},
			Rules: []PolicyRule{
				{
					Verbs:     []string{"get"},
					Resources: []string{"pods"},
				},
			},
		},
		wantErr: false,
	}.test(t)
}

//func TestValidateRoleNoAPIGroup(t *testing.T) {
//	ValidateRoleTest{
//		role: Role{
//			ObjectMeta: api.ObjectMeta{
//				Name:      "default",
//				Namespace: "default",
//			},
//			Rules: []PolicyRule{
//				{
//					Verbs:     []string{"get"},
//					Resources: []string{"pods"},
//				},
//			},
//		},
//		wantErr: true,
//		errType: field.ErrorTypeRequired,
//		field:   "rules[0].apiGroups",
//	}.test(t)
//}

func TestValidateRoleNoResources(t *testing.T) {
	ValidateRoleTest{
		role: Role{
			ObjectMeta: api.ObjectMeta{
				Name:      "default",
				Namespace: "default",
			},
			Rules: []PolicyRule{
				{
					Verbs: []string{"get"},
				},
			},
		},
		wantErr: true,
		errType: field.ErrorTypeRequired,
		field:   "rules[0].resources",
	}.test(t)
}
