package cmdcli

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	utiltesting "github.com/yubo/apiserver/pkg/rest/testing"
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
	expectedBody, _ := runtime.Encode(scheme.Codecs.LegacyCodec(), output)
	fakeHandler := utiltesting.FakeHandler{
		StatusCode:   200,
		ResponseBody: string(expectedBody),
		T:            t,
	}
	testServer := httptest.NewServer(&fakeHandler)
	return testServer, &fakeHandler
}
