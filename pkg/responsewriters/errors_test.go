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

package responsewriters

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/yubo/apiserver/pkg/authentication/user"
	"github.com/yubo/apiserver/pkg/authorization/authorizer"
	"github.com/yubo/apiserver/pkg/request"
	"github.com/yubo/golib/scheme"
)

func TestErrors(t *testing.T) {
	internalError := errors.New("ARGH")
	fns := map[string]func(http.ResponseWriter, *http.Request){
		"InternalError": func(w http.ResponseWriter, req *http.Request) {
			InternalError(w, req, internalError)
		},
	}
	cases := []struct {
		fn       string
		uri      string
		expected string
	}{
		{"InternalError", "/get", "Internal Server Error: \"/get\": ARGH\n"},
		{"InternalError", "/<script>", "Internal Server Error: \"/&lt;script&gt;\": ARGH\n"},
	}
	for _, test := range cases {
		observer := httptest.NewRecorder()
		fns[test.fn](observer, &http.Request{RequestURI: test.uri})
		result := string(observer.Body.Bytes())
		if result != test.expected {
			t.Errorf("%s(..., %q) != %q, got %q", test.fn, test.uri, test.expected, result)
		}
	}
}

func TestForbidden(t *testing.T) {
	u := &user.DefaultInfo{Name: "NAME"}
	cases := []struct {
		expected    string
		attributes  authorizer.Attributes
		reason      string
		contentType string
	}{
		{`{"metadata":{},"status":"Failure","message":"forbidden: User \"NAME\" cannot GET path \"/whatever\"","reason":"Forbidden","details":{},"code":403}
`, authorizer.AttributesRecord{User: u, Verb: "GET", Path: "/whatever"}, "", "application/json"},
		{`{"metadata":{},"status":"Failure","message":"forbidden: User \"NAME\" cannot GET path \"/\u0026lt;script\u0026gt;\"","reason":"Forbidden","details":{},"code":403}
`, authorizer.AttributesRecord{User: u, Verb: "GET", Path: "/<script>"}, "", "application/json"},
		{`{"metadata":{},"status":"Failure","message":"forbidden: User \"NAME\" cannot get resource \"pod\" at the cluster scope","reason":"Forbidden","details":{},"code":403}
`, authorizer.AttributesRecord{User: u, Verb: "get", Resource: "pod", ResourceRequest: true}, "", "application/json"},
		{`{"metadata":{},"status":"Failure","message":"forbidden: User \"NAME\" cannot get resource \"pod\" at the cluster scope","reason":"Forbidden","details":{"name":"mypod"},"code":403}
`, authorizer.AttributesRecord{User: u, Verb: "get", Resource: "pod", ResourceRequest: true, Name: "mypod"}, "", "application/json"},
		{`{"metadata":{},"status":"Failure","message":"forbidden: User \"NAME\" cannot get resource \"pod/quota\" in the namespace \"test\"","reason":"Forbidden","details":{},"code":403}
`, authorizer.AttributesRecord{User: u, Verb: "get", Namespace: "test", APIGroup: "v2", Resource: "pod", Subresource: "quota", ResourceRequest: true}, "", "application/json"},
	}
	for _, test := range cases {
		observer := httptest.NewRecorder()
		negotiatedSerializer := scheme.NegotiatedSerializer
		Forbidden(request.NewDefaultContext(), test.attributes, observer, &http.Request{URL: &url.URL{Path: "/path"}}, test.reason, negotiatedSerializer)
		result := string(observer.Body.Bytes())
		if result != test.expected {
			t.Errorf("Forbidden response body(%#v...)\n expected: %v\ngot:       %v", test.attributes, test.expected, result)
		}
		resultType := observer.HeaderMap.Get("Content-Type")
		if resultType != test.contentType {
			t.Errorf("Forbidden content type(%#v...) != %#v, got %#v", test.attributes, test.expected, result)
		}
	}
}
