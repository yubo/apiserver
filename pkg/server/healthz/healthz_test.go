/*
Copyright 2014 The Kubernetes Authors.

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

package healthz

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/yubo/apiserver/pkg/metrics"
	"github.com/yubo/golib/api"
	"github.com/yubo/golib/util/sets"
)

func TestInstallHandler(t *testing.T) {
	mux := http.NewServeMux()
	InstallHandler(mux)
	req, err := http.NewRequest("GET", "http://example.com/healthz", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected %v, got %v", http.StatusOK, w.Code)
	}
	c := w.Header().Get("Content-Type")
	if c != "text/plain; charset=utf-8" {
		t.Errorf("expected %v, got %v", "text/plain", c)
	}
	if w.Body.String() != "ok" {
		t.Errorf("expected %v, got %v", "ok", w.Body.String())
	}
}

func TestInstallPathHandler(t *testing.T) {
	mux := http.NewServeMux()
	InstallPathHandler(mux, "/healthz/test")
	InstallPathHandler(mux, "/healthz/ready")
	req, err := http.NewRequest("GET", "http://example.com/healthz/test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected %v, got %v", http.StatusOK, w.Code)
	}
	c := w.Header().Get("Content-Type")
	if c != "text/plain; charset=utf-8" {
		t.Errorf("expected %v, got %v", "text/plain", c)
	}
	if w.Body.String() != "ok" {
		t.Errorf("expected %v, got %v", "ok", w.Body.String())
	}

	req, err = http.NewRequest("GET", "http://example.com/healthz/ready", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected %v, got %v", http.StatusOK, w.Code)
	}
	c = w.Header().Get("Content-Type")
	if c != "text/plain; charset=utf-8" {
		t.Errorf("expected %v, got %v", "text/plain", c)
	}
	if w.Body.String() != "ok" {
		t.Errorf("expected %v, got %v", "ok", w.Body.String())
	}

}

func testMultipleChecks(path, name string, t *testing.T) {
	tests := []struct {
		path             string
		expectedResponse string
		expectedStatus   int
		addBadCheck      bool
	}{
		{"?verbose", fmt.Sprintf("[+]ping ok\n%s check passed\n", name), http.StatusOK, false},
		{"?exclude=dontexist", "ok", http.StatusOK, false},
		{"?exclude=bad", "ok", http.StatusOK, true},
		{"?verbose=true&exclude=bad", fmt.Sprintf("[+]ping ok\n[+]bad excluded: ok\n%s check passed\n", name), http.StatusOK, true},
		{"?verbose=true&exclude=dontexist", fmt.Sprintf("[+]ping ok\nwarn: some health checks cannot be excluded: no matches for \"dontexist\"\n%s check passed\n", name), http.StatusOK, false},
		{"/ping", "ok", http.StatusOK, false},
		{"", "ok", http.StatusOK, false},
		{"?verbose", fmt.Sprintf("[+]ping ok\n[-]bad failed: reason withheld\n%s check failed\n", name), http.StatusInternalServerError, true},
		{"/ping", "ok", http.StatusOK, true},
		{"/bad", "internal server error: this will fail\n", http.StatusInternalServerError, true},
		{"", fmt.Sprintf("[+]ping ok\n[-]bad failed: reason withheld\n%s check failed\n", name), http.StatusInternalServerError, true},
	}

	for i, test := range tests {
		mux := http.NewServeMux()
		checks := []HealthChecker{PingHealthz}
		if test.addBadCheck {
			checks = append(checks, NamedCheck("bad", func(_ *http.Request) error {
				return errors.New("this will fail")
			}))
		}
		if path == "" {
			InstallHandler(mux, checks...)
			path = "/healthz"
		} else {
			InstallPathHandler(mux, path, checks...)
		}
		req, err := http.NewRequest("GET", fmt.Sprintf("http://example.com%s%v", path, test.path), nil)
		if err != nil {
			t.Fatalf("case[%d] Unexpected error: %v", i, err)
		}
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		if w.Code != test.expectedStatus {
			t.Errorf("case[%d] Expected: %v, got: %v", i, test.expectedStatus, w.Code)
		}
		c := w.Header().Get("Content-Type")
		if c != "text/plain; charset=utf-8" {
			t.Errorf("case[%d] Expected: %v, got: %v", i, "text/plain", c)
		}
		if w.Body.String() != test.expectedResponse {
			t.Errorf("case[%d] Expected:\n%v\ngot:\n%v\n", i, test.expectedResponse, w.Body.String())
		}
	}
}

func TestMultipleChecks(t *testing.T) {
	testMultipleChecks("", "healthz", t)
}

func TestMultiplePathChecks(t *testing.T) {
	testMultipleChecks("/ready", "ready", t)
}

func TestCheckerNames(t *testing.T) {
	n1 := "n1"
	n2 := "n2"
	c1 := &healthzCheck{name: n1}
	c2 := &healthzCheck{name: n2}

	testCases := []struct {
		desc string
		have []HealthChecker
		want []string
	}{
		{"no checker", []HealthChecker{}, []string{}},
		{"one checker", []HealthChecker{c1}, []string{n1}},
		{"other checker", []HealthChecker{c2}, []string{n2}},
		{"checker order", []HealthChecker{c1, c2}, []string{n1, n2}},
		{"different checker order", []HealthChecker{c2, c1}, []string{n2, n1}},
	}

	for _, tc := range testCases {
		result := checkerNames(tc.have...)
		t.Run(tc.desc, func(t *testing.T) {
			if !reflect.DeepEqual(tc.want, result) {
				t.Errorf("want %#v, got %#v", tc.want, result)
			}
		})
	}
}

func TestFormatQuoted(t *testing.T) {
	n1 := "n1"
	n2 := "n2"
	testCases := []struct {
		desc     string
		names    []string
		expected string
	}{
		{"empty", []string{}, ""},
		{"single name", []string{n1}, "\"n1\""},
		{"two names", []string{n1, n2}, "\"n1\",\"n2\""},
		{"two names, reverse order", []string{n2, n1}, "\"n2\",\"n1\""},
	}
	for _, tc := range testCases {
		result := formatQuoted(tc.names...)
		t.Run(tc.desc, func(t *testing.T) {
			if result != tc.expected {
				t.Errorf("expected %#v, got %#v", tc.expected, result)
			}
		})
	}
}

func TestGetExcludedChecks(t *testing.T) {
	tests := []struct {
		name string
		r    *http.Request
		want sets.String
	}{
		{"Should have no excluded health checks",
			createGetRequestWithUrl("/healthz?verbose=true"),
			sets.NewString(),
		},
		{"Should extract out the ping health check",
			createGetRequestWithUrl("/healthz?exclude=ping"),
			sets.NewString("ping"),
		},
		{"Should extract out ping and log health check",
			createGetRequestWithUrl("/healthz?exclude=ping&exclude=log"),
			sets.NewString("ping", "log"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getExcludedChecks(tt.r); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getExcludedChecks() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMetrics(t *testing.T) {
	mux := http.NewServeMux()
	InstallHandler(mux)
	InstallLivezHandler(mux)
	InstallReadyzHandler(mux)
	metrics.Register()
	metrics.Reset()

	paths := []string{"/healthz", "/livez", "/readyz"}
	for _, path := range paths {
		req, err := http.NewRequest("GET", fmt.Sprintf("http://example.com%s", path), nil)
		if err != nil {
			t.Errorf("%v", err)
		}
		mux.ServeHTTP(httptest.NewRecorder(), req)
	}

	expected := strings.NewReader(`
        # HELP apiserver_request_total Counter of apiserver requests broken out for each verb, dry run value, path, component, and HTTP response contentType and code.
        # TYPE apiserver_request_total counter
        apiserver_request_total{code="200",component="",contentType="text/plain;charset=utf-8",dry_run="",path="/healthz",verb="GET"} 1
        apiserver_request_total{code="200",component="",contentType="text/plain;charset=utf-8",dry_run="",path="/livez",verb="GET"} 1
        apiserver_request_total{code="200",component="",contentType="text/plain;charset=utf-8",dry_run="",path="/readyz",verb="GET"} 1
`)
	if err := testutil.GatherAndCompare(prometheus.DefaultGatherer, expected, "apiserver_request_total"); err != nil {
		t.Error(err)
	}
}

func createGetRequestWithUrl(rawUrlString string) *http.Request {
	url, _ := url.Parse(rawUrlString)
	return &http.Request{
		Method: http.MethodGet,
		Proto:  "HTTP/1.1",
		URL:    url,
	}
}

func TestInformerSyncHealthChecker(t *testing.T) {
	t.Run("test that check returns nil when all informers are started", func(t *testing.T) {
		healthChecker := NewInformerSyncHealthz(cacheSyncWaiterStub{
			startedByInformerType: map[reflect.Type]bool{
				reflect.TypeOf(api.Event{}): true,
			},
		})

		err := healthChecker.Check(nil)
		if err != nil {
			t.Errorf("Got %v, expected no error", err)
		}
	})

	t.Run("test that check returns err when there is not started informer", func(t *testing.T) {
		healthChecker := NewInformerSyncHealthz(cacheSyncWaiterStub{
			startedByInformerType: map[reflect.Type]bool{
				reflect.TypeOf(api.Event{}):    true,
				reflect.TypeOf(api.UserInfo{}): false,
				reflect.TypeOf(api.Secret{}):   true,
			},
		})

		err := healthChecker.Check(nil)
		if err == nil {
			t.Errorf("expected error, got: %v", err)
		}
	})
}

type cacheSyncWaiterStub struct {
	startedByInformerType map[reflect.Type]bool
}

// WaitForCacheSync is a stub implementation of the corresponding func
// that simply returns the value passed during stub initialization.
func (s cacheSyncWaiterStub) WaitForCacheSync(_ <-chan struct{}) map[reflect.Type]bool {
	return s.startedByInformerType
}
