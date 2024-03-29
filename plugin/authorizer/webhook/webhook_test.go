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

package webhook

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/stretchr/testify/assert"
	authorization "github.com/yubo/apiserver/pkg/apis/authorization"
	"github.com/yubo/apiserver/pkg/authentication/user"
	"github.com/yubo/apiserver/pkg/authorization/authorizer"
	v1 "github.com/yubo/client-go/tools/clientcmd/api/v1"
	"github.com/yubo/golib/api"
	"github.com/yubo/golib/util/wait"
)

var testRetryBackoff = wait.Backoff{
	Duration: api.NewDuration("5ms"),
	Factor:   1.5,
	Jitter:   0.2,
	Steps:    5,
}

func TestV1NewFromConfig(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	data := struct {
		CA   string
		Cert string
		Key  string
	}{
		CA:   filepath.Join(dir, "ca.pem"),
		Cert: filepath.Join(dir, "clientcert.pem"),
		Key:  filepath.Join(dir, "clientkey.pem"),
	}

	files := []struct {
		name string
		data []byte
	}{
		{data.CA, caCert},
		{data.Cert, clientCert},
		{data.Key, clientKey},
	}
	for _, file := range files {
		if err := ioutil.WriteFile(file.name, file.data, 0400); err != nil {
			t.Fatal(err)
		}
	}

	tests := []struct {
		msg        string
		configTmpl string
		wantErr    bool
	}{
		{
			msg: "a single cluster and single user",
			configTmpl: `
clusters:
- cluster:
    certificate-authority: {{ .CA }}
    server: https://authz.example.com
  name: foobar
users:
- name: a cluster
  user:
    client-certificate: {{ .Cert }}
    client-key: {{ .Key }}
`,
			wantErr: true,
		},
		{
			msg: "multiple clusters with no context",
			configTmpl: `
clusters:
- cluster:
    certificate-authority: {{ .CA }}
    server: https://authz.example.com
  name: foobar
- cluster:
    certificate-authority: a bad certificate path
    server: https://authz.example.com
  name: barfoo
users:
- name: a name
  user:
    client-certificate: {{ .Cert }}
    client-key: {{ .Key }}
`,
			wantErr: true,
		},
		{
			msg: "multiple clusters with a context",
			configTmpl: `
clusters:
- cluster:
    certificate-authority: a bad certificate path
    server: https://authz.example.com
  name: foobar
- cluster:
    certificate-authority: {{ .CA }}
    server: https://authz.example.com
  name: barfoo
users:
- name: a name
  user:
    client-certificate: {{ .Cert }}
    client-key: {{ .Key }}
contexts:
- name: default
  context:
    cluster: barfoo
    user: a name
current-context: default
`,
			wantErr: false,
		},
		{
			msg: "cluster with bad certificate path specified",
			configTmpl: `
clusters:
- cluster:
    certificate-authority: a bad certificate path
    server: https://authz.example.com
  name: foobar
- cluster:
    certificate-authority: {{ .CA }}
    server: https://authz.example.com
  name: barfoo
users:
- name: a name
  user:
    client-certificate: {{ .Cert }}
    client-key: {{ .Key }}
contexts:
- name: default
  context:
    cluster: foobar
    user: a name
current-context: default
`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		// Use a closure so defer statements trigger between loop iterations.
		err := func() error {
			tempfile, err := ioutil.TempFile("", "")
			if err != nil {
				return err
			}
			p := tempfile.Name()
			defer os.Remove(p)

			tmpl, err := template.New("test").Parse(tt.configTmpl)
			if err != nil {
				return fmt.Errorf("failed to parse test template: %v", err)
			}
			if err := tmpl.Execute(tempfile, data); err != nil {
				return fmt.Errorf("failed to execute test template: %v", err)
			}
			// Create a new authorizer
			sarClient, err := subjectAccessReviewInterfaceFromKubeconfig(p, testRetryBackoff, nil)
			if err != nil {
				return fmt.Errorf("error building sar client: %v", err)
			}
			_, err = newWithBackoff(sarClient, 0, 0, testRetryBackoff)
			return err
		}()
		if err != nil && !tt.wantErr {
			t.Errorf("failed to load plugin from config %q: %v", tt.msg, err)
		}
		if err == nil && tt.wantErr {
			t.Errorf("wanted an error when loading config, did not get one: %q", tt.msg)
		}
	}
}

// V1Service mocks a remote service.
type V1Service interface {
	Review(*authorization.SubjectAccessReview)
	HTTPStatusCode() int
}

// NewV1TestServer wraps a V1Service as an httptest.Server.
func NewV1TestServer(s V1Service, cert, key, caCert []byte) (*httptest.Server, error) {
	const webhookPath = "/testserver"
	var tlsConfig *tls.Config
	if cert != nil {
		cert, err := tls.X509KeyPair(cert, key)
		if err != nil {
			return nil, err
		}
		tlsConfig = &tls.Config{Certificates: []tls.Certificate{cert}}
	}

	if caCert != nil {
		rootCAs := x509.NewCertPool()
		rootCAs.AppendCertsFromPEM(caCert)
		if tlsConfig == nil {
			tlsConfig = &tls.Config{}
		}
		tlsConfig.ClientCAs = rootCAs
		tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
	}

	serveHTTP := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, fmt.Sprintf("unexpected method: %v", r.Method), http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Path != webhookPath {
			http.Error(w, fmt.Sprintf("unexpected path: %v", r.URL.Path), http.StatusNotFound)
			return
		}

		var review authorization.SubjectAccessReview
		bodyData, _ := ioutil.ReadAll(r.Body)
		if err := json.Unmarshal(bodyData, &review); err != nil {
			http.Error(w, fmt.Sprintf("failed to decode body: %v", err), http.StatusBadRequest)
			return
		}

		// ensure we received the serialized review as expected
		//if review.APIVersion != "authorization.k8s.io/v1" {
		//	http.Error(w, fmt.Sprintf("wrong api version: %s", string(bodyData)), http.StatusBadRequest)
		//	return
		//}
		// once we have a successful request, always call the review to record that we were called
		s.Review(&review)
		if s.HTTPStatusCode() < 200 || s.HTTPStatusCode() >= 300 {
			http.Error(w, "HTTP Error", s.HTTPStatusCode())
			return
		}
		type status struct {
			Allowed         bool   `json:"allowed"`
			Reason          string `json:"reason"`
			EvaluationError string `json:"evaluationError"`
		}
		resp := struct {
			//APIVersion string `json:"apiVersion"`
			Status status `json:"status"`
		}{
			//APIVersion: authorization.SchemeGroupVersion.String(),
			Status: status{review.Status.Allowed, review.Status.Reason, review.Status.EvaluationError},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}

	server := httptest.NewUnstartedServer(http.HandlerFunc(serveHTTP))
	server.TLS = tlsConfig
	server.StartTLS()

	// Adjust the path to point to our custom path
	serverURL, _ := url.Parse(server.URL)
	serverURL.Path = webhookPath
	server.URL = serverURL.String()

	return server, nil
}

// A service that can be set to allow all or deny all authorization requests.
type mockV1Service struct {
	allow      bool
	statusCode int
	called     int
}

func (m *mockV1Service) Review(r *authorization.SubjectAccessReview) {
	m.called++
	r.Status.Allowed = m.allow
}
func (m *mockV1Service) Allow()              { m.allow = true }
func (m *mockV1Service) Deny()               { m.allow = false }
func (m *mockV1Service) HTTPStatusCode() int { return m.statusCode }

// newV1Authorizer creates a temporary kubeconfig file from the provided arguments and attempts to load
// a new WebhookAuthorizer from it.
func newV1Authorizer(callbackURL string, clientCert, clientKey, ca []byte, cacheTime time.Duration) (*WebhookAuthorizer, error) {
	tempfile, err := ioutil.TempFile("", "")
	if err != nil {
		return nil, err
	}
	p := tempfile.Name()
	defer os.Remove(p)
	config := v1.Config{
		Clusters: []v1.NamedCluster{
			{
				Cluster: v1.Cluster{Server: callbackURL, CertificateAuthorityData: ca},
			},
		},
		AuthInfos: []v1.NamedAuthInfo{
			{
				AuthInfo: v1.AuthInfo{ClientCertificateData: clientCert, ClientKeyData: clientKey},
			},
		},
	}
	if err := json.NewEncoder(tempfile).Encode(config); err != nil {
		return nil, err
	}
	sarClient, err := subjectAccessReviewInterfaceFromKubeconfig(p, testRetryBackoff, nil)
	if err != nil {
		return nil, fmt.Errorf("error building sar client: %v", err)
	}
	return newWithBackoff(sarClient, cacheTime, cacheTime, testRetryBackoff)
}

func TestV1TLSConfig(t *testing.T) {
	tests := []struct {
		test                            string
		clientCert, clientKey, clientCA []byte
		serverCert, serverKey, serverCA []byte
		wantAuth, wantErr               bool
	}{
		{
			test:       "TLS setup between client and server",
			clientCert: clientCert, clientKey: clientKey, clientCA: caCert,
			serverCert: serverCert, serverKey: serverKey, serverCA: caCert,
			wantAuth: true,
		},
		{
			test:       "Server does not require client auth",
			clientCA:   caCert,
			serverCert: serverCert, serverKey: serverKey,
			wantAuth: true,
		},
		{
			test:       "Server does not require client auth, client provides it",
			clientCert: clientCert, clientKey: clientKey, clientCA: caCert,
			serverCert: serverCert, serverKey: serverKey,
			wantAuth: true,
		},
		{
			test:       "Client does not trust server",
			clientCert: clientCert, clientKey: clientKey,
			serverCert: serverCert, serverKey: serverKey,
			wantErr: true,
		},
		{
			test:       "Server does not trust client",
			clientCert: clientCert, clientKey: clientKey, clientCA: caCert,
			serverCert: serverCert, serverKey: serverKey, serverCA: badCACert,
			wantErr: true,
		},
		{
			// Plugin does not support insecure configurations.
			test:    "Server is using insecure connection",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		// Use a closure so defer statements trigger between loop iterations.
		func() {
			service := new(mockV1Service)
			service.statusCode = 200

			server, err := NewV1TestServer(service, tt.serverCert, tt.serverKey, tt.serverCA)
			if err != nil {
				t.Errorf("%s: failed to create server: %v", tt.test, err)
				return
			}
			defer server.Close()

			wh, err := newV1Authorizer(server.URL, tt.clientCert, tt.clientKey, tt.clientCA, 0)
			if err != nil {
				t.Errorf("%s: failed to create client: %v", tt.test, err)
				return
			}

			attr := authorizer.AttributesRecord{User: &user.DefaultInfo{}}

			// Allow all and see if we get an error.
			service.Allow()
			decision, _, err := wh.Authorize(context.Background(), attr)
			if tt.wantAuth {
				if decision != authorizer.DecisionAllow {
					t.Errorf("expected successful authorization")
				}
			} else {
				if decision == authorizer.DecisionAllow {
					t.Errorf("expected failed authorization")
				}
			}
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error making authorization request: %v", err)
				}
				return
			}
			if err != nil {
				t.Errorf("%s: failed to authorize with AllowAll policy: %v", tt.test, err)
				return
			}

			service.Deny()
			if decision, _, _ := wh.Authorize(context.Background(), attr); decision == authorizer.DecisionAllow {
				t.Errorf("%s: incorrectly authorized with DenyAll policy", tt.test)
			}
		}()
	}
}

// recorderV1Service records all access review requests.
type recorderV1Service struct {
	last authorization.SubjectAccessReview
	err  error
}

func (rec *recorderV1Service) Review(r *authorization.SubjectAccessReview) {
	rec.last = authorization.SubjectAccessReview{}
	rec.last = *r
	r.Status.Allowed = true
}

func (rec *recorderV1Service) Last() (authorization.SubjectAccessReview, error) {
	return rec.last, rec.err
}

func (rec *recorderV1Service) HTTPStatusCode() int { return 200 }

func TestV1Webhook(t *testing.T) {
	serv := new(recorderV1Service)
	s, err := NewV1TestServer(serv, serverCert, serverKey, caCert)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	wh, err := newV1Authorizer(s.URL, clientCert, clientKey, caCert, 0)
	if err != nil {
		t.Fatal(err)
	}

	expTypeMeta := api.TypeMeta{
		//APIVersion: "authorization.k8s.io/v1",
		//Kind:       "SubjectAccessReview",
	}

	tests := []struct {
		attr authorizer.Attributes
		want authorization.SubjectAccessReview
	}{
		{
			attr: authorizer.AttributesRecord{User: &user.DefaultInfo{}},
			want: authorization.SubjectAccessReview{
				TypeMeta: expTypeMeta,
				Spec: authorization.SubjectAccessReviewSpec{
					NonResourceAttributes: &authorization.NonResourceAttributes{},
				},
			},
		},
		{
			attr: authorizer.AttributesRecord{User: &user.DefaultInfo{Name: "jane"}},
			want: authorization.SubjectAccessReview{
				TypeMeta: expTypeMeta,
				Spec: authorization.SubjectAccessReviewSpec{
					User:                  "jane",
					NonResourceAttributes: &authorization.NonResourceAttributes{},
				},
			},
		},
		{
			attr: authorizer.AttributesRecord{
				User: &user.DefaultInfo{
					Name:   "jane",
					UID:    "1",
					Groups: []string{"group1", "group2"},
				},
				Verb:            "GET",
				Namespace:       "kittensandponies",
				APIGroup:        "group3",
				APIVersion:      "v7beta3",
				Resource:        "pods",
				Subresource:     "proxy",
				Name:            "my-pod",
				ResourceRequest: true,
				Path:            "/foo",
			},
			want: authorization.SubjectAccessReview{
				TypeMeta: expTypeMeta,
				Spec: authorization.SubjectAccessReviewSpec{
					User:   "jane",
					UID:    "1",
					Groups: []string{"group1", "group2"},
					ResourceAttributes: &authorization.ResourceAttributes{
						Verb:        "GET",
						Namespace:   "kittensandponies",
						Group:       "group3",
						Version:     "v7beta3",
						Resource:    "pods",
						Subresource: "proxy",
						Name:        "my-pod",
					},
				},
			},
		},
	}

	for i, tt := range tests {
		decision, _, err := wh.Authorize(context.Background(), tt.attr)
		if err != nil {
			t.Fatal(err)
		}
		if decision != authorizer.DecisionAllow {
			t.Errorf("case %d: authorization failed", i)
			continue
		}

		gotAttr, err := serv.Last()
		if err != nil {
			t.Errorf("case %d: failed to deserialize webhook request: %v", i, err)
			continue
		}
		assert.Equal(t, gotAttr, tt.want)
	}
}

// TestWebhookCache verifies that error responses from the server are not
// cached, but successful responses are.
func TestV1WebhookCache(t *testing.T) {
	serv := new(mockV1Service)
	s, err := NewV1TestServer(serv, serverCert, serverKey, caCert)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	// Create an authorizer that caches successful responses "forever" (100 days).
	wh, err := newV1Authorizer(s.URL, clientCert, clientKey, caCert, 2400*time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	aliceAttr := authorizer.AttributesRecord{User: &user.DefaultInfo{Name: "alice"}}
	bobAttr := authorizer.AttributesRecord{User: &user.DefaultInfo{Name: "bob"}}
	aliceRidiculousAttr := authorizer.AttributesRecord{
		User:            &user.DefaultInfo{Name: "alice"},
		ResourceRequest: true,
		Verb:            strings.Repeat("v", 2000),
		APIGroup:        strings.Repeat("g", 2000),
		APIVersion:      strings.Repeat("a", 2000),
		Resource:        strings.Repeat("r", 2000),
		Name:            strings.Repeat("n", 2000),
	}
	bobRidiculousAttr := authorizer.AttributesRecord{
		User:            &user.DefaultInfo{Name: "bob"},
		ResourceRequest: true,
		Verb:            strings.Repeat("v", 2000),
		APIGroup:        strings.Repeat("g", 2000),
		APIVersion:      strings.Repeat("a", 2000),
		Resource:        strings.Repeat("r", 2000),
		Name:            strings.Repeat("n", 2000),
	}

	type webhookCacheTestCase struct {
		name string

		attr authorizer.AttributesRecord

		allow      bool
		statusCode int

		expectedErr        bool
		expectedAuthorized bool
		expectedCalls      int
	}

	tests := []webhookCacheTestCase{
		// server error and 429's retry
		{name: "server errors retry", attr: aliceAttr, allow: false, statusCode: 500, expectedErr: true, expectedAuthorized: false, expectedCalls: 5},
		{name: "429s retry", attr: aliceAttr, allow: false, statusCode: 429, expectedErr: true, expectedAuthorized: false, expectedCalls: 5},
		// regular errors return errors but do not retry
		{name: "404 doesnt retry", attr: aliceAttr, allow: false, statusCode: 404, expectedErr: true, expectedAuthorized: false, expectedCalls: 1},
		{name: "403 doesnt retry", attr: aliceAttr, allow: false, statusCode: 403, expectedErr: true, expectedAuthorized: false, expectedCalls: 1},
		{name: "401 doesnt retry", attr: aliceAttr, allow: false, statusCode: 401, expectedErr: true, expectedAuthorized: false, expectedCalls: 1},
		// successful responses are cached
		{name: "alice successful request", attr: aliceAttr, allow: true, statusCode: 200, expectedErr: false, expectedAuthorized: true, expectedCalls: 1},
		// later requests within the cache window don't hit the backend
		{name: "alice cached request", attr: aliceAttr, allow: false, statusCode: 500, expectedErr: false, expectedAuthorized: true, expectedCalls: 0},

		// a request with different attributes doesn't hit the cache
		{name: "bob failed request", attr: bobAttr, allow: false, statusCode: 500, expectedErr: true, expectedAuthorized: false, expectedCalls: 5},
		// successful response for other attributes is cached
		{name: "bob unauthorized request", attr: bobAttr, allow: false, statusCode: 200, expectedErr: false, expectedAuthorized: false, expectedCalls: 1},
		// later requests within the cache window don't hit the backend
		{name: "bob unauthorized cached request", attr: bobAttr, allow: false, statusCode: 500, expectedErr: false, expectedAuthorized: false, expectedCalls: 0},
		// ridiculous unauthorized requests are not cached.
		{name: "ridiculous unauthorized request", attr: bobRidiculousAttr, allow: false, statusCode: 200, expectedErr: false, expectedAuthorized: false, expectedCalls: 1},
		// later ridiculous requests within the cache window still hit the backend
		{name: "ridiculous unauthorized request again", attr: bobRidiculousAttr, allow: false, statusCode: 200, expectedErr: false, expectedAuthorized: false, expectedCalls: 1},
		// ridiculous authorized requests are not cached.
		{name: "ridiculous authorized request", attr: aliceRidiculousAttr, allow: true, statusCode: 200, expectedErr: false, expectedAuthorized: true, expectedCalls: 1},
		// later ridiculous requests within the cache window still hit the backend
		{name: "ridiculous authorized request again", attr: aliceRidiculousAttr, allow: true, statusCode: 200, expectedErr: false, expectedAuthorized: true, expectedCalls: 1},
	}

	for i, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			serv.called = 0
			serv.allow = test.allow
			serv.statusCode = test.statusCode
			authorized, _, err := wh.Authorize(context.Background(), test.attr)
			if test.expectedErr && err == nil {
				t.Fatalf("%d: Expected error", i)
			} else if !test.expectedErr && err != nil {
				t.Fatalf("%d: unexpected error: %v", i, err)
			}

			if test.expectedAuthorized != (authorized == authorizer.DecisionAllow) {
				t.Errorf("%d: expected authorized=%v, got %v", i, test.expectedAuthorized, authorized)
			}

			if test.expectedCalls != serv.called {
				t.Errorf("%d: expected %d calls, got %d", i, test.expectedCalls, serv.called)
			}
		})
	}
}
