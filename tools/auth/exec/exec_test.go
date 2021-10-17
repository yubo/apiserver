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
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	clientauthentication "github.com/yubo/apiserver/pkg/apis/clientauthentication"
	"github.com/yubo/apiserver/pkg/rest"
	"github.com/yubo/golib/api"
	"github.com/yubo/golib/runtime"
)

// restInfo holds the rest.Client fields that we care about for test assertions.
type restInfo struct {
	host            string
	tlsClientConfig rest.TLSClientConfig
	proxyURL        string
}

func TestLoadExecCredential(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name               string
		data               []byte
		wantExecCredential runtime.Object
		wantRESTInfo       restInfo
		wantErrorPrefix    string
	}{
		{
			name: "v1beta1 happy path",
			data: marshal(t, &clientauthentication.ExecCredential{
				Spec: clientauthentication.ExecCredentialSpec{
					Cluster: &clientauthentication.Cluster{
						Server:                   "https://some-server/some/path",
						TLSServerName:            "some-server-name",
						InsecureSkipTLSVerify:    true,
						CertificateAuthorityData: []byte("some-ca-data"),
						ProxyURL:                 "https://some-proxy-url:12345",
						//Config: runtime.RawExtension{
						//	Raw: []byte(`{"apiVersion":"group/v1","kind":"PluginConfig","spec":{"names":["marshmallow","zelda"]}}`),
						//},
					},
				},
			}),
			wantExecCredential: &clientauthentication.ExecCredential{
				TypeMeta: api.TypeMeta{
					//Kind: "ExecCredential",
				},
				Spec: clientauthentication.ExecCredentialSpec{
					Cluster: &clientauthentication.Cluster{
						Server:                   "https://some-server/some/path",
						TLSServerName:            "some-server-name",
						InsecureSkipTLSVerify:    true,
						CertificateAuthorityData: []byte("some-ca-data"),
						ProxyURL:                 "https://some-proxy-url:12345",
						//Config: runtime.RawExtension{
						//	Raw: []byte(`{"apiVersion":"group/v1","kind":"PluginConfig","spec":{"names":["marshmallow","zelda"]}}`),
						//},
					},
				},
			},
			wantRESTInfo: restInfo{
				host: "https://some-server/some/path",
				tlsClientConfig: rest.TLSClientConfig{
					Insecure:   true,
					ServerName: "some-server-name",
					CAData:     []byte("some-ca-data"),
				},
				proxyURL: "https://some-proxy-url:12345",
			},
		},
		{
			name: "v1beta1 nil config",
			data: marshal(t, &clientauthentication.ExecCredential{
				Spec: clientauthentication.ExecCredentialSpec{
					Cluster: &clientauthentication.Cluster{
						Server:                   "https://some-server/some/path",
						TLSServerName:            "some-server-name",
						InsecureSkipTLSVerify:    true,
						CertificateAuthorityData: []byte("some-ca-data"),
						ProxyURL:                 "https://some-proxy-url:12345",
					},
				},
			}),
			wantExecCredential: &clientauthentication.ExecCredential{
				TypeMeta: api.TypeMeta{
					//Kind: "ExecCredential",
				},
				Spec: clientauthentication.ExecCredentialSpec{
					Cluster: &clientauthentication.Cluster{
						Server:                   "https://some-server/some/path",
						TLSServerName:            "some-server-name",
						InsecureSkipTLSVerify:    true,
						CertificateAuthorityData: []byte("some-ca-data"),
						ProxyURL:                 "https://some-proxy-url:12345",
					},
				},
			},
			wantRESTInfo: restInfo{
				host: "https://some-server/some/path",
				tlsClientConfig: rest.TLSClientConfig{
					Insecure:   true,
					ServerName: "some-server-name",
					CAData:     []byte("some-ca-data"),
				},
				proxyURL: "https://some-proxy-url:12345",
			},
		},
		{
			name: "v1beta1 invalid cluster",
			data: marshal(t, &clientauthentication.ExecCredential{
				Spec: clientauthentication.ExecCredentialSpec{
					Cluster: &clientauthentication.Cluster{
						ProxyURL: "invalid- url\n",
					},
				},
			}),
			wantErrorPrefix: "cannot create rest.Config",
		},
		{
			name:            "v1beta1 nil cluster",
			data:            marshal(t, &clientauthentication.ExecCredential{}),
			wantErrorPrefix: "ExecCredential does not contain cluster information",
		},
		//{
		//	name:            "invalid object kind",
		//	data:            marshal(t, &api.Status{}),
		//	wantErrorPrefix: "invalid group/kind: wanted ExecCredential.client.authentication.k8s.io, got Status",
		//},
		{
			name:            "bad data",
			data:            []byte("bad data"),
			wantErrorPrefix: "decode: ",
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			execCredential, restConfig, err := LoadExecCredential(test.data)
			if test.wantErrorPrefix != "" {
				if err == nil {
					t.Error("wanted error, got success")
				} else if !strings.HasPrefix(err.Error(), test.wantErrorPrefix) {
					t.Errorf("wanted '%s', got '%s'", test.wantErrorPrefix, err.Error())
				}
			} else if err != nil {
				t.Errorf("%s, %s", test.name, err)
			} else {
				if diff := cmp.Diff(test.wantExecCredential, execCredential); diff != "" {
					t.Error(diff)
				}

				if diff := cmp.Diff(test.wantRESTInfo.host, restConfig.Host); diff != "" {
					t.Error(diff)
				}
				if diff := cmp.Diff(test.wantRESTInfo.tlsClientConfig, restConfig.TLSClientConfig); diff != "" {
					t.Error(diff)
				}

				proxyURL, err := restConfig.Proxy(nil)
				if err != nil {
					t.Fatal(err)
				}
				if diff := cmp.Diff(test.wantRESTInfo.proxyURL, proxyURL.String()); diff != "" {
					t.Error(diff)
				}
			}
		})
	}
}

func marshal(t *testing.T, obj runtime.Object) []byte {
	t.Helper()

	data, err := runtime.Encode(codecs.LegacyCodec(), obj)
	if err != nil {
		t.Fatal(err)
	}

	return data
}
