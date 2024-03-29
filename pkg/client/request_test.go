package client

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	utiltesting "github.com/yubo/client-go/util/testing"
	"github.com/yubo/golib/runtime"
	"github.com/yubo/golib/scheme"
)

type TestParam struct {
	actualError    error
	expectingError bool
	actualBody     interface{}
	expectingBody  interface{}
}

type Foo struct {
	Bar int
}

var (
	input  = &Foo{1}
	output = &Foo{2}
)

func TestRequest(t *testing.T) {
	cases := []struct {
		method string
		input  interface{}
		output interface{}
	}{
		{"GET", nil, nil},
		{"POST", nil, nil},
		{"PUT", nil, nil},
		{"DELETE", nil, nil},
		{"GET", input, output},
		{"POST", input, output},
		{"PUT", input, output},
		{"DELETE", input, output},
	}

	for _, c := range cases {
		t.Run(c.method, func(t *testing.T) {
			testServer, fakeHandler := testServerEnv(t, c.output)
			defer testServer.Close()

			actualOutput := &Foo{}
			opts := []RequestOption{
				WithMethod("GET"),
				WithPrefix("test"),
				WithBody(c.input),
			}
			if c.output != nil {
				opts = append(opts, WithOutput(actualOutput))
			}

			req, err := NewRequest(testServer.URL, opts...)
			assert.NoError(t, err)

			err = req.Do(context.Background())
			assert.NoError(t, err)

			if c.output != nil {
				assert.Equal(t, c.output, actualOutput)
			}

			validate(t, "GET", "/test", c.input, fakeHandler)
		})
	}
}

func TestRequestParam(t *testing.T) {
	type Foo struct {
		Namespace string `param:"path" name:"namespace"`
		Current   int    `param:"query" name:"current"`
		PageSize  int    `param:"query" name:"pageSize"`
	}

	testServer, fakeHandler := testServerEnv(t, nil)
	defer testServer.Close()

	req, err := NewRequest(
		testServer.URL,
		WithMethod("GET"),
		WithPath("/api/v1/{namespace}"),
		WithParams(&Foo{
			Namespace: "test",
			Current:   1,
			PageSize:  10,
		}),
	)
	assert.NoError(t, err)

	err = req.Do(context.Background())
	assert.NoError(t, err)

	assert.Equal(t, "/api/v1/test?current=1&pageSize=10", fakeHandler.RequestReceived.RequestURI)

}

func validate(t *testing.T, expectedMethod, expectedPath string, expectedInput interface{}, fakeHandler *utiltesting.FakeHandler) {
	if expectedInput == nil {
		fakeHandler.ValidateRequest(t, expectedPath, expectedMethod, nil)
		return
	}

	buff, _ := runtime.Encode(scheme.Codecs.LegacyCodec(), expectedInput)
	body := string(buff)
	fakeHandler.ValidateRequest(t, expectedPath, expectedMethod, &body)
}

func testServerEnv(t *testing.T, output interface{}) (*httptest.Server, *utiltesting.FakeHandler) {
	expectedBody, _ := runtime.Encode(scheme.Codec, output)
	fakeHandler := utiltesting.FakeHandler{
		StatusCode:   200,
		ResponseBody: string(expectedBody),
		T:            t,
	}
	testServer := httptest.NewServer(&fakeHandler)
	return testServer, &fakeHandler
}
