package rest

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/yubo/golib/scheme"
	utiltesting "github.com/yubo/apiserver/pkg/rest/testing"
	"github.com/yubo/golib/runtime"
	"github.com/yubo/golib/api"
	"github.com/yubo/golib/api/errors"
	"github.com/yubo/golib/util/diff"
	"k8s.io/klog/v2"
)

type TestParam struct {
	actualError           error
	expectingError        bool
	actualCreated         bool
	expCreated            bool
	expStatus             *api.Status
	testBody              bool
	testBodyErrorIsNotNil bool
}

func TestDoRequestSuccess(t *testing.T) {
	testServer, fakeHandler, status := testServerEnv(t, 200)
	defer testServer.Close()

	c, err := restClient(testServer)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	body, err := c.Get().Prefix("test").Do(context.Background()).Raw()

	testParam := TestParam{actualError: err, expectingError: false, expCreated: true,
		expStatus: status, testBody: true, testBodyErrorIsNotNil: false}
	validate(testParam, t, body, fakeHandler)
}

func TestDoRequestFailed(t *testing.T) {
	status := &api.Status{
		Code:    http.StatusNotFound,
		Status:  api.StatusFailure,
		Reason:  api.StatusReasonNotFound,
		Message: "the server could not find the requested resource",
		Details: &api.StatusDetails{},
	}
	expectedBody, _ := runtime.Encode(scheme.Codecs.LegacyCodec(), status)
	klog.Infof("-- %v", string(expectedBody))
	fakeHandler := utiltesting.FakeHandler{
		StatusCode:   404,
		ResponseBody: string(expectedBody),
		T:            t,
	}
	testServer := httptest.NewServer(&fakeHandler)
	defer testServer.Close()

	c, err := restClient(testServer)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	err = c.Get().Do(context.Background()).Error()
	if err == nil {
		t.Errorf("unexpected non-error")
	}
	ss, ok := err.(errors.APIStatus)
	if !ok {
		t.Errorf("unexpected error type %v", err)
	}
	actual := ss.Status()
	if !reflect.DeepEqual(status, &actual) {
		t.Errorf("Unexpected mis-match: %s", diff.ObjectReflectDiff(status, &actual))
	}
}

func TestDoRawRequestFailed(t *testing.T) {
	status := &api.Status{
		Code:    http.StatusNotFound,
		Status:  api.StatusFailure,
		Reason:  api.StatusReasonNotFound,
		Message: "the server could not find the requested resource",
		Details: &api.StatusDetails{
			Causes: []api.StatusCause{
				{Type: api.CauseTypeUnexpectedServerResponse, Message: "unknown"},
			},
		},
	}
	expectedBody, _ := runtime.Encode(scheme.Codecs.LegacyCodec(), status)
	fakeHandler := utiltesting.FakeHandler{
		StatusCode:   404,
		ResponseBody: string(expectedBody),
		T:            t,
	}
	testServer := httptest.NewServer(&fakeHandler)
	defer testServer.Close()

	c, err := restClient(testServer)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	body, err := c.Get().Do(context.Background()).Raw()

	if err == nil || body == nil {
		t.Errorf("unexpected non-error: %#v", body)
	}
	ss, ok := err.(errors.APIStatus)
	if !ok {
		t.Errorf("unexpected error type %v", err)
	}
	actual := ss.Status()
	if !reflect.DeepEqual(status, &actual) {
		t.Errorf("Unexpected mis-match: %s", diff.ObjectReflectDiff(status, &actual))
	}
}

func TestDoRequestCreated(t *testing.T) {
	testServer, fakeHandler, status := testServerEnv(t, http.StatusCreated)
	defer testServer.Close()

	c, err := restClient(testServer)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	created := false
	body, err := c.Get().Prefix("test").Do(context.Background()).WasCreated(&created).Raw()

	testParam := TestParam{actualError: err, expectingError: false, expCreated: true,
		expStatus: status, testBody: false}
	validate(testParam, t, body, fakeHandler)
}

func TestDoRequestNotCreated(t *testing.T) {
	testServer, fakeHandler, expectedStatus := testServerEnv(t, 202)
	defer testServer.Close()
	c, err := restClient(testServer)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	created := false
	body, err := c.Get().Prefix("test").Do(context.Background()).WasCreated(&created).Raw()
	testParam := TestParam{actualError: err, expectingError: false, expCreated: false,
		expStatus: expectedStatus, testBody: false}
	validate(testParam, t, body, fakeHandler)
}

func TestDoRequestAcceptedNoContentReturned(t *testing.T) {
	testServer, fakeHandler, _ := testServerEnv(t, 204)
	defer testServer.Close()

	c, err := restClient(testServer)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	created := false
	body, err := c.Get().Prefix("test").Do(context.Background()).WasCreated(&created).Raw()
	testParam := TestParam{actualError: err, expectingError: false, expCreated: false,
		testBody: false}
	validate(testParam, t, body, fakeHandler)
}

func TestBadRequest(t *testing.T) {
	testServer, fakeHandler, _ := testServerEnv(t, 400)
	defer testServer.Close()
	c, err := restClient(testServer)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	created := false
	body, err := c.Get().Prefix("test").Do(context.Background()).WasCreated(&created).Raw()
	testParam := TestParam{actualError: err, expectingError: true, expCreated: false,
		testBody: true}
	validate(testParam, t, body, fakeHandler)
}

func validate(testParam TestParam, t *testing.T, body []byte, fakeHandler *utiltesting.FakeHandler) {
	switch {
	case testParam.expectingError && testParam.actualError == nil:
		t.Errorf("Expected error")
	case !testParam.expectingError && testParam.actualError != nil:
		t.Error(testParam.actualError)
	}
	if !testParam.expCreated {
		if testParam.actualCreated {
			t.Errorf("Expected object not to be created")
		}
	}
	statusOut := &api.Status{}
	_, err := runtime.Decode(scheme.Codecs.UniversalDeserializer(), body, statusOut)
	if testParam.testBody {
		if testParam.testBodyErrorIsNotNil && err == nil {
			t.Errorf("Expected Error")
		}
		if !testParam.testBodyErrorIsNotNil && err != nil {
			t.Errorf("Unexpected Error: %v", err)
		}
	}

	if testParam.expStatus != nil {
		if !reflect.DeepEqual(testParam.expStatus, statusOut) {
			t.Errorf("Unexpected mis-match. Expected %#v.  Saw %#v", testParam.expStatus, statusOut)
		}
	}
	fakeHandler.ValidateRequest(t, "/test", "GET", nil)

}

func testServerEnv(t *testing.T, statusCode int) (*httptest.Server, *utiltesting.FakeHandler, *api.Status) {
	status := &api.Status{TypeMeta: api.TypeMeta{APIVersion: "v1", Kind: "Status"}, Status: fmt.Sprintf("%s", api.StatusSuccess)}
	expectedBody, _ := runtime.Encode(scheme.Codecs.LegacyCodec(), status)
	fakeHandler := utiltesting.FakeHandler{
		StatusCode:   statusCode,
		ResponseBody: string(expectedBody),
		T:            t,
	}
	testServer := httptest.NewServer(&fakeHandler)
	return testServer, &fakeHandler, status
}

func restClient(testServer *httptest.Server) (*RESTClient, error) {
	c, err := RESTClientFor(&Config{
		Host: testServer.URL,
		ContentConfig: ContentConfig{
			NegotiatedSerializer: scheme.Codecs.WithoutConversion(),
		},
		Username: "user",
		Password: "pass",
	})
	return c, err
}
