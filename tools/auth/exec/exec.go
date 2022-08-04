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

// Package exec contains helper utilities for exec credential plugins.
package exec

import (
	"errors"
	"fmt"
	"os"

	"github.com/yubo/apiserver/pkg/apis/clientauthentication"
	rest "github.com/yubo/apiserver/pkg/client"
	"github.com/yubo/golib/runtime"
	"github.com/yubo/golib/runtime/serializer"
)

const execInfoEnv = "KUBERNETES_EXEC_INFO"

//var scheme = runtime.NewScheme()
var codecs = serializer.NewCodecFactory()

//func init() {
//	metav1.AddToGroupVersion(scheme, schema.GroupVersion{Version: "v1"})
//	utilruntime.Must(v1alpha1.AddToScheme(scheme))
//	utilruntime.Must(v1beta1.AddToScheme(scheme))
//	utilruntime.Must(clientauthentication.AddToScheme(scheme))
//}

// LoadExecCredentialFromEnv is a helper-wrapper around LoadExecCredential that loads from the
// well-known KUBERNETES_EXEC_INFO environment variable.
//
// When the KUBERNETES_EXEC_INFO environment variable is not set or is empty, then this function
// will immediately return an error.
func LoadExecCredentialFromEnv() (runtime.Object, *rest.Config, error) {
	env := os.Getenv(execInfoEnv)
	if env == "" {
		return nil, nil, errors.New("KUBERNETES_EXEC_INFO env var is unset or empty")
	}
	return LoadExecCredential([]byte(env))
}

// LoadExecCredential loads the configuration needed for an exec plugin to communicate with a
// cluster.
//
// LoadExecCredential expects the provided data to be a serialized client.authentication.k8s.io
// ExecCredential object (of any version). If the provided data is invalid (i.e., it cannot be
// unmarshalled into any known client.authentication.k8s.io ExecCredential version), an error will
// be returned. A successfully unmarshalled ExecCredential will be returned as the first return
// value.
//
// If the provided data is successfully unmarshalled, but it does not contain cluster information
// (i.e., ExecCredential.Spec.Cluster == nil), then the returned rest.Config and error will be nil.
//
// Note that the returned rest.Config will use anonymous authentication, since the exec plugin has
// not returned credentials for this cluster yet.
func LoadExecCredential(data []byte) (runtime.Object, *rest.Config, error) {
	obj := &clientauthentication.ExecCredential{}
	_, err := codecs.UniversalDeserializer().Decode(data, obj)
	if err != nil {
		return nil, nil, fmt.Errorf("decode: %w", err)
	}

	if obj.Spec.Cluster == nil {
		return nil, nil, errors.New("ExecCredential does not contain cluster information")
	}

	restConfig, err := rest.ExecClusterToConfig(obj.Spec.Cluster)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot create rest.Config: %w", err)
	}

	return obj, restConfig, nil
}
