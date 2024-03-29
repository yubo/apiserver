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

package clientauthentication

import (
	"github.com/yubo/golib/api"
	"github.com/yubo/golib/runtime"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ExecCredential is used by exec-based plugins to communicate credentials to
// HTTP transports.
type ExecCredential struct {
	api.TypeMeta

	// Spec holds information passed to the plugin by the transport. This contains
	// request and runtime specific information, such as if the session is interactive.
	Spec ExecCredentialSpec `json:"spec,omitempty"`

	// Status is filled in by the plugin and holds the credentials that the transport
	// should use to contact the API.
	// +optional
	Status *ExecCredentialStatus `json:"status,omitempty"`
}

// ExecCredentialSpec holds request and runtime specific information provided by
// the transport.
type ExecCredentialSpec struct {
	// Response is populated when the transport encounters HTTP status codes, such as 401,
	// suggesting previous credentials were invalid.
	// +optional
	Response *Response `json:"response,omitempty"`

	// Interactive is true when the transport detects the command is being called from an
	// interactive prompt.
	// +optional
	Interactive bool `json:"interactive,omitempty"`

	// Cluster contains information to allow an exec plugin to communicate with the
	// kubernetes cluster being authenticated to. Note that Cluster is non-nil only
	// when provideClusterInfo is set to true in the exec provider config (i.e.,
	// ExecConfig.ProvideClusterInfo).
	// +optional
	Cluster *Cluster `json:"cluster,omitempty"`
}

// ExecCredentialStatus holds credentials for the transport to use.
type ExecCredentialStatus struct {
	// ExpirationTimestamp indicates a time when the provided credentials expire.
	// +optional
	ExpirationTimestamp *api.Time `json:"expirationTimestamp,omitempty"`
	// Token is a bearer token used by the client for request authentication.
	// +optional
	Token string `json:"token,omitempty" datapolicy:"token"`
	// PEM-encoded client TLS certificate.
	// +optional
	ClientCertificateData string `json:"clientCertificateData,omitempty"`
	// PEM-encoded client TLS private key.
	// +optional
	ClientKeyData string `json:"clientKeyData,omitempty" datapolicy:"secret-key"`
}

// Response defines metadata about a failed request, including HTTP status code and
// response headers.
type Response struct {
	// Headers holds HTTP headers returned by the server.
	Header map[string][]string `json:"header,omitempty"`
	// Code is the HTTP status code returned by the server.
	Code int32 `json:"code,omitempty"`
}

// Cluster contains information to allow an exec plugin to communicate
// with the kubernetes cluster being authenticated to.
//
// To ensure that this struct contains everything someone would need to communicate
// with a kubernetes cluster (just like they would via a kubeconfig), the fields
// should shadow "github.com/yubo/client-go/tools/clientcmd/api/v1".Cluster, with the exception
// of CertificateAuthority, since CA data will always be passed to the plugin as bytes.
type Cluster struct {
	// Server is the address of the kubernetes cluster (https://hostname:port).
	Server string `json:"server,omitempty"`
	// TLSServerName is passed to the server for SNI and is used in the client to
	// check server certificates against. If ServerName is empty, the hostname
	// used to contact the server is used.
	// +optional
	TLSServerName string `json:"tls-server-name,omitempty"`
	// InsecureSkipTLSVerify skips the validity check for the server's certificate.
	// This will make your HTTPS connections insecure.
	// +optional
	InsecureSkipTLSVerify bool `json:"insecure-skip-tls-verify,omitempty"`
	// CAData contains PEM-encoded certificate authority certificates.
	// If empty, system roots should be used.
	// +listType=atomic
	// +optional
	CertificateAuthorityData []byte `json:"certificate-authority-data,omitempty"`
	// ProxyURL is the URL to the proxy to be used for all requests to this
	// cluster.
	// +optional
	ProxyURL string `json:"proxy-url,omitempty"`
	// Config holds additional config data that is specific to the exec
	// plugin with regards to the cluster being authenticated to.
	//
	// This data is sourced from the clientcmd Cluster object's
	// extensions[client.authentication.k8s.io/exec] field:
	//
	// clusters:
	// - name: my-cluster
	//   cluster:
	//     ...
	//     extensions:
	//     - name: client.authentication.k8s.io/exec  # reserved extension name for per cluster exec config
	//       extension:
	//         audience: 06e3fbd18de8  # arbitrary config
	//
	// In some environments, the user config may be exactly the same across many clusters
	// (i.e. call this exec plugin) minus some details that are specific to each cluster
	// such as the audience.  This field allows the per cluster config to be directly
	// specified with the cluster info.  Using this field to store secret data is not
	// recommended as one of the prime benefits of exec plugins is that no secrets need
	// to be stored directly in the kubeconfig.
	// +optional
	Config runtime.Object `json:"config,omitempty"`
}
