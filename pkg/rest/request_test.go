package rest

import (
	"net/http"
	"strings"
	"testing"

	"github.com/emicklei/go-restful"
	"github.com/stretchr/testify/require"
	"github.com/yubo/golib/util"
	"k8s.io/klog/v2"
)

type SampleStruct struct {
	StructValue *string `json:"structValue"`
	foo         *string
	bar         string
}

type Sample struct {
	PathValue   *string `param:"path" name:"name"`
	HeaderValue *string `param:"header" name:"headerValue"`
	QueryValue  *string `param:"query" name:"queryValue"`
	foo         *string
	bar         string
}

type SampleBody struct {
	DataValueString *string       `json:"dataValueString"`
	DataValueStruct *SampleStruct `json:"dataValueStruct"`
}

func init() {
	var level klog.Level
	level.Set("20")
}

func TestRequest(t *testing.T) {
	opt := &RequestOptions{
		Url:    "http://example.com/users/{name}",
		Method: "GET",
		InputParam: &Sample{
			PathValue:   util.String("tom"),
			HeaderValue: util.String("HeaderValue"),
			QueryValue:  util.String("QueryValue"),
		},
		InputBody: &SampleBody{
			DataValueString: util.String("DataValueString"),
			DataValueStruct: &SampleStruct{
				StructValue: util.String("StructValue"),
			},
		},
	}

	if _, err := NewRequest(opt); err != nil {
		t.Fatal(err)
	}
}

/*
func TestJson(t *testing.T) {
	json.Unmarshal()
	json.Marshal()
}
*/

func TestRequestEncode(t *testing.T) {
	header0 := make(http.Header)
	header0.Set("Accept", "*/*")
	header1 := make(http.Header)
	header1.Set("Accept", "*/*")
	header1.Set("headerValue", "HeaderValue")

	cases := []struct {
		url        string
		inputParam *Sample
		wantUrl    string
		wantHeader http.Header
	}{{
		"",
		&Sample{},
		"",
		header0,
	}, {
		"http://example.com/users/{name}",
		&Sample{PathValue: util.String("tom")},
		"http://example.com/users/tom",
		header0,
	}, {
		"",
		&Sample{HeaderValue: util.String("HeaderValue")},
		"",
		header1,
	}, {
		"",
		&Sample{QueryValue: util.String("QueryValue")},
		"?queryValue=QueryValue",
		header0,
	}, {
		"",
		&Sample{},
		"",
		header0,
	}, {
		"http://example.com/users/{name}",
		&Sample{
			PathValue:   util.String("tom"),
			HeaderValue: util.String("HeaderValue"),
			QueryValue:  util.String("QueryValue"),
		},
		"http://example.com/users/tom?queryValue=QueryValue",
		header1,
	}}

	for i, c := range cases {
		req, err := NewRequest(&RequestOptions{Url: c.url, InputParam: c.inputParam})
		require.Emptyf(t, err, "case-%d", i)

		err = req.prepareParam()
		require.Emptyf(t, err, "case-%d", i)
		require.Equalf(t, c.wantUrl, req.url, "cases-%d", i)
		require.Equalf(t, c.wantHeader, req.header, "cases-%d", i)
	}
}

func TestInvokePathVariable(t *testing.T) {
	data := map[string]string{
		"user-name":   "tom",
		"user-id":     "16",
		"api-version": "1",
		"empty":       "",
	}

	cases := []struct {
		in   string
		want string
	}{
		{"{user-name}", "tom"},
		{"/{user-name}", "/tom"},
		{"{user-name}/", "tom/"},
		{"/{empty}/a", "//a"},
		{"/{user-name}/{user-id}/", "/tom/16/"},
		{"http://example.com/api/v{api-version}/user/{user-id}",
			"http://example.com/api/v1/user/16"},
	}

	for i, c := range cases {
		got, err := invokePathVariable(c.in, data)
		require.Emptyf(t, err, "case-%d", i)
		require.Equalf(t, c.want, got, "case-%d", i)
	}
}

func TestReadEntity(t *testing.T) {
	header0 := make(http.Header)
	header1 := make(http.Header)
	header1.Set("headerValue", "HeaderValue")

	cases := []struct {
		url       string
		body      string
		header    http.Header
		wantParam *Sample
		wantBody  *SampleBody
	}{{
		"",
		"{}",
		header0,
		&Sample{},
		&SampleBody{},
	}, {
		"",
		"{}",
		header1,
		&Sample{
			HeaderValue: util.String("HeaderValue"),
		},
		&SampleBody{},
	}, {
		"?queryValue=QueryValue",
		"{}",
		header0,
		&Sample{
			QueryValue: util.String("QueryValue"),
		},
		&SampleBody{},
	}, {
		"",
		`{"dataValueString" : "DataValueString"}`,
		header0,
		&Sample{},
		&SampleBody{
			DataValueString: util.String("DataValueString"),
		},
	}, {
		"",
		`{
			"dataValueStruct": {"structValue": "StructValue"}
		}`,
		header0,
		&Sample{},
		&SampleBody{
			DataValueStruct: &SampleStruct{
				StructValue: util.String("StructValue"),
			},
		},
	}, {
		"?queryValue=QueryValue",
		`{
			"dataValueString" : "DataValueString" ,
			"dataValueStruct": {"structValue": "StructValue"}
		}`,
		header1,
		&Sample{
			HeaderValue: util.String("HeaderValue"),
			QueryValue:  util.String("QueryValue"),
		},
		&SampleBody{
			DataValueString: util.String("DataValueString"),
			DataValueStruct: &SampleStruct{
				StructValue: util.String("StructValue"),
			},
		},
	}}

	for i, c := range cases {
		httpRequest, _ := http.NewRequest("GET", c.url, strings.NewReader(c.body))
		httpRequest.Header = c.header
		httpRequest.Header.Set("Content-Type", "application/json")
		request := restful.NewRequest(httpRequest)

		gotParam := &Sample{}
		gotBody := &SampleBody{}
		if err := ReadEntity(request, gotParam, gotBody); err != nil {
			t.Fatalf("case-%d ReadEntity failed %v", i, err)
		}

		require.Equalf(t, c.wantParam, gotParam, "case-%d", i)
		require.Equalf(t, c.wantBody, gotBody, "case-%d", i)
	}
}
