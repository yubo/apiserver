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

package reference

import (
	"github.com/yubo/golib/api"
)

type TestRuntimeObj struct {
	api.TypeMeta
	api.ObjectMeta
}

//func (o *TestRuntimeObj) DeepCopyObject() runtime.Object {
//	panic("die")
//}
//
//func TestGetReferenceRefVersion(t *testing.T) {
//	tests := []struct {
//		name               string
//		input              *TestRuntimeObj
//		expectedRefVersion string
//	}{
//		{
//			name: "v1 GV from scheme",
//			input: &TestRuntimeObj{
//				ObjectMeta: api.ObjectMeta{SelfLink: "/bad-selflink/unused"},
//			},
//			expectedRefVersion: "v1",
//		},
//		{
//			name: "foo.group/v3 GV from scheme",
//			input: &TestRuntimeObj{
//				ObjectMeta: api.ObjectMeta{SelfLink: "/bad-selflink/unused"},
//			},
//			expectedRefVersion: "foo.group/v3",
//		},
//	}
//
//	for _, test := range tests {
//		t.Run(test.name, func(t *testing.T) {
//			ref, err := GetReference(test.input)
//			if err != nil {
//				t.Fatal(err)
//			}
//			if test.expectedRefVersion != ref.APIVersion {
//				t.Errorf("expected %q, got %q", test.expectedRefVersion, ref.APIVersion)
//			}
//		})
//	}
//}
