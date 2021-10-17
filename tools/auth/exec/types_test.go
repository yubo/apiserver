/*
Copyright 2020 The Kubernetes Authors.

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

package exec

import (
	"reflect"
	"testing"

	"github.com/yubo/apiserver/pkg/apis/clientauthentication"
	clientcmdapi "github.com/yubo/apiserver/tools/clientcmd/api"
	"github.com/yubo/golib/util/sets"
)

// TestV1beta1ClusterTypesAreSynced ensures that clientauthentication.Cluster stays in sync
// with clientcmdapi.Cluster.
//
// We want clientauthentication.Cluster to offer the same knobs as clientcmdapi.Cluster to
// allow someone to connect to the kubernetes API. This test should fail if a new field is added to
// one of the structs without updating the other.
func TestClusterTypesAreSynced(t *testing.T) {
	t.Parallel()

	execType := reflect.TypeOf(clientauthentication.Cluster{})
	clientcmdType := reflect.TypeOf(clientcmdapi.Cluster{})

	t.Run("exec cluster fields match clientcmd cluster fields", func(t *testing.T) {
		t.Parallel()

		// These are fields that are specific to Cluster and shouldn't be in clientcmdapi.Cluster.
		execSkippedFieldNames := sets.NewString(
			// Cluster uses Config to provide its cluster-specific configuration object.
			"Config",
		)

		for i := 0; i < execType.NumField(); i++ {
			execField := execType.Field(i)
			if execSkippedFieldNames.Has(execField.Name) {
				continue
			}

			t.Run(execField.Name, func(t *testing.T) {
				t.Parallel()
				clientcmdField, ok := clientcmdType.FieldByName(execField.Name)
				if !ok {
					t.Errorf("unknown field (please add field to clientcmdapi.Cluster): '%s'", execField.Name)
				} else if execField.Type != clientcmdField.Type {
					t.Errorf(
						"type mismatch (please update Cluster.%s field type to match clientcmdapi.Cluster.%s field type): %q != %q",
						execField.Name,
						clientcmdField.Name,
						execField.Type,
						clientcmdField.Type,
					)
				} else if execField.Tag != clientcmdField.Tag {
					t.Errorf(
						"tag mismatch (please update Cluster.%s tag to match clientcmdapi.Cluster.%s tag): %q != %q",
						execField.Name,
						clientcmdField.Name,
						execField.Tag,
						clientcmdField.Tag,
					)
				}
			})
		}
	})

	t.Run("clientcmd cluster fields match exec cluster fields", func(t *testing.T) {
		t.Parallel()

		// These are the fields that we don't want to shadow from clientcmdapi.Cluster.
		clientcmdSkippedFieldNames := sets.NewString(
			"LocationOfOrigin",
			// CA data will be passed via CertificateAuthorityData, so we don't need this field.
			"CertificateAuthority",
			// Cluster uses Config to provide its cluster-specific configuration object.
			"Extensions",
		)

		for i := 0; i < clientcmdType.NumField(); i++ {
			clientcmdField := clientcmdType.Field(i)
			if clientcmdSkippedFieldNames.Has(clientcmdField.Name) {
				continue
			}

			t.Run(clientcmdField.Name, func(t *testing.T) {
				t.Parallel()
				execField, ok := execType.FieldByName(clientcmdField.Name)
				if !ok {
					t.Errorf("unknown field (please add field to Cluster): '%s'", clientcmdField.Name)
				} else if clientcmdField.Type != execField.Type {
					t.Errorf(
						"type mismatch (please update clientcmdapi.Cluster.%s field type to match Cluster.%s field type): %q != %q",
						clientcmdField.Name,
						execField.Name,
						clientcmdField.Type,
						execField.Type,
					)
				} else if clientcmdField.Tag != execField.Tag {
					t.Errorf(
						"tag mismatch (please update clientcmdapi.Cluster.%s tag to match Cluster.%s tag): %q != %q",
						clientcmdField.Name,
						execField.Name,
						clientcmdField.Tag,
						execField.Tag,
					)
				}
			})
		}
	})
}

// TestAllClusterTypesAreSynced is a TODO so that we remember to write a test similar to
// TestV1beta1ClusterTypesAreSynced for any future ExecCredential version. It should start failing
// when someone adds support for any other ExecCredential type to this package.
//func TestAllClusterTypesAreSynced(t *testing.T) {
//	versionsThatDontNeedTests := sets.NewString(
//		// The internal Cluster type should only be used...internally...and therefore doesn't
//		// necessarily need to be synced with clientcmdapi.
//		runtime.APIVersionInternal,
//		// V1alpha1 does not contain a Cluster type.
//		clientauthenticationv1alpha1.SchemeGroupVersion.Version,
//		// We have a test for v1beta1 above.
//		clientauthenticationv1beta1.SchemeGroupVersion.Version,
//	)
//	for gvk := range scheme.AllKnownTypes() {
//		if gvk.Group == clientauthenticationv1beta1.SchemeGroupVersion.Group &&
//			gvk.Kind == "ExecCredential" {
//			if !versionsThatDontNeedTests.Has(gvk.Version) {
//				t.Errorf(
//					"TODO: add test similar to TestV1beta1ClusterTypesAreSynced for client.authentication.k8s.io/%s",
//					gvk.Version,
//				)
//			}
//		}
//	}
//}
