package rest

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/emicklei/go-restful"
	"github.com/stretchr/testify/require"
	"github.com/yubo/golib/util"
	"k8s.io/klog/v2"
)

func tearDown() {
	restful.DefaultContainer = restful.NewContainer()
}

func dbgFilter(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
	klog.V(8).Infof("tracing filter entering")
	if klog.V(8).Enabled() {
		klog.Infof("[req] HTTP %v %v", req.Request.Method, req.SelectedRoutePath())

		body, _ := ioutil.ReadAll(req.Request.Body)
		if len(body) > 0 {
			req.Request.Body = ioutil.NopCloser(bytes.NewBuffer(body))
		}
		klog.Info("[req] " + Req2curl(req.Request, body, nil, nil))
	}
	chain.ProcessFilter(req, resp)
	if klog.V(8).Enabled() {
		err := resp.Error()
		if resp.StatusCode() == http.StatusFound {
			klog.Infof("[resp] %v %v %d %s", req.Request.Method, req.SelectedRoutePath(), resp.StatusCode(), resp.Header().Get("location"))
		} else {
			klog.Infof("[resp] %v %v %d %v", req.Request.Method, req.SelectedRoutePath(), resp.StatusCode(), err)
		}
	}
}

func TestHttpParam(t *testing.T) {
	type GetHost struct {
		Dir   *string `param:"path"`
		Name  *string `param:"path"`
		Query *string `param:"query"`
	}

	type GetHostBody struct {
		Ip  *string `json:"ip"`
		Dns *string `json:"dns"`
	}

	cases := []struct {
		param *GetHost
		body  *GetHostBody
	}{{
		&GetHost{
			Dir:   util.String("dir"),
			Name:  util.String("name"),
			Query: util.String("query"),
		},
		&GetHostBody{
			Ip:  util.String("1.1.1.1"),
			Dns: util.String("jbo-ep-dev.fw"),
		},
	}, {
		&GetHost{
			Dir:  util.String("dir"),
			Name: util.String("name"),
		},
		&GetHostBody{
			Ip:  util.String(""),
			Dns: util.String(""),
		},
	}, {
		&GetHost{
			Dir:  util.String("dir"),
			Name: util.String("name"),
		},
		&GetHostBody{},
	}}

	for i, c := range cases {
		tearDown()
		ws := new(restful.WebService).Path("").Consumes("MIME_JSON")
		ws.Route(ws.POST("/dirs/{dir}/hosts/{name}").
			Consumes(MIME_JSON).
			Produces(MIME_JSON).
			Filter(dbgFilter).
			To(func(req *restful.Request, resp *restful.Response) {
				param := GetHost{}
				body := GetHostBody{}
				err := ReadEntity(req, &param, &body)
				require.Emptyf(t, err, "case-%d", i)
				require.Equalf(t, c.param, &param, "case-%d", i)
				require.Equalf(t, c.body, &body, "case-%d", i)
			}))
		restful.DefaultContainer.Add(ws)

		// write
		opt := &RequestOptions{
			Method:     "POST",
			Url:        "http://example/dirs/{dir}/hosts/{name}",
			InputParam: c.param,
			InputBody:  c.body,
		}
		req, err := NewRequest(opt)
		if err != nil {
			t.Fatal(err)
		}
		require.Emptyf(t, err, "case-%d", i)

		httpWriter := httptest.NewRecorder()
		restful.DefaultContainer.DoNotRecover(false)
		restful.DefaultContainer.Dispatch(httpWriter, req.Request)

		require.Equalf(t, 200, httpWriter.Code, "case-%d %s %s", i, req, httpWriter.Body.String())

	}
}

func TestWsRouteBuild(t *testing.T) {
	type UpdateInput struct {
		Namespace string `param:"path"`
		Name      string `param:"path"`
	}

	type UpdateUserInput struct {
		Age         *int    `json:"ip"`
		DisplayName *string `json:"displayName"`
	}

	type User struct {
		Name        *string `json:"name"`
		Age         *int    `json:"ip"`
		DisplayName *string `json:"displayName"`
	}

	{
		cases := []interface{}{}
		cases = append(cases, &UpdateInput{Namespace: "default", Name: "tom"},
			&UpdateUserInput{Age: util.Int(26), DisplayName: util.String("sonic2020")})

		for i := 0; i < len(cases)/2; i++ {
			tearDown()

			param := cases[2*i]
			body := cases[2*i+1]
			ws := new(restful.WebService).Path("").Consumes("MIME_JSON")
			WsRouteBuild(&WsOption{
				Ws: ws.Path("").Produces(MIME_JSON).Consumes(MIME_JSON),
			}, []WsRoute{{
				Method: "PUT", SubPath: "/namespaces/{namespace}/users/{name}",
				Desc: "update user",
				Handle: func(req *restful.Request, resp *restful.Response, p *UpdateInput, b *UpdateUserInput) {
					require.Equalf(t, param, p, "case-%d", i)
					require.Equalf(t, body, b, "case-%d", i)
					return
				},
			}})
			restful.DefaultContainer.Add(ws)

			// write
			opt := &RequestOptions{
				Method:     "PUT",
				Url:        "http://example/namespaces/{namespace}/users/{name}",
				InputParam: param,
				InputBody:  body,
			}
			req, err := NewRequest(opt)
			require.Emptyf(t, err, "case-%d", i)

			httpWriter := httptest.NewRecorder()
			restful.DefaultContainer.DoNotRecover(false)
			restful.DefaultContainer.Dispatch(httpWriter, req.Request)
			require.Equalf(t, 200, httpWriter.Code, "case-%d %s %s", i, req, httpWriter.Body.String())
		}
	}

	{
		cases := []interface{}{}
		cases = append(cases, &UpdateInput{Namespace: "default", Name: "tom"},
			[]UpdateUserInput{{Age: util.Int(26), DisplayName: util.String("sonic2020")}})

		for i := 0; i < len(cases)/2; i++ {
			tearDown()

			param := cases[2*i]
			body := cases[2*i+1]
			ws := new(restful.WebService).Path("").Consumes("MIME_JSON")
			WsRouteBuild(&WsOption{
				Ws: ws.Path("").Produces(MIME_JSON).Consumes(MIME_JSON),
			}, []WsRoute{{
				Method: "PUT", SubPath: "/namespaces/{namespace}/users/{name}",
				Desc: "update user",
				Handle: func(req *restful.Request, resp *restful.Response, p *UpdateInput, b []UpdateUserInput) {
					require.Equalf(t, param, p, "case-%d", i)
					require.Equalf(t, body, b, "case-%d", i)
					return
				},
			}})
			restful.DefaultContainer.Add(ws)

			// write
			opt := &RequestOptions{
				Method:     "PUT",
				Url:        "http://example/namespaces/{namespace}/users/{name}",
				InputParam: param,
				InputBody:  body,
			}
			req, err := NewRequest(opt)
			require.Emptyf(t, err, "case-%d", i)

			httpWriter := httptest.NewRecorder()
			restful.DefaultContainer.DoNotRecover(false)
			restful.DefaultContainer.Dispatch(httpWriter, req.Request)
			require.Equalf(t, 200, httpWriter.Code, "case-%d %s %s", i, req, httpWriter.Body.String())
		}
	}

	{
		cases := []interface{}{}
		cases = append(cases, &UpdateInput{Namespace: "default", Name: "tom"},
			[]*UpdateUserInput{&UpdateUserInput{Age: util.Int(26), DisplayName: util.String("sonic2020")}})

		for i := 0; i < len(cases)/2; i++ {
			tearDown()

			param := cases[2*i]
			body := cases[2*i+1]
			ws := new(restful.WebService).Path("").Consumes("MIME_JSON")
			WsRouteBuild(&WsOption{
				Ws: ws.Path("").Produces(MIME_JSON).Consumes(MIME_JSON),
			}, []WsRoute{{
				Method: "PUT", SubPath: "/namespaces/{namespace}/users/{name}",
				Desc: "update user",
				Handle: func(req *restful.Request, resp *restful.Response, p *UpdateInput, b []*UpdateUserInput) {
					require.Equalf(t, param, p, "case-%d", i)
					require.Equalf(t, body, b, "case-%d", i)
					return
				},
			}})
			restful.DefaultContainer.Add(ws)

			// write
			opt := &RequestOptions{
				Method:     "PUT",
				Url:        "http://example/namespaces/{namespace}/users/{name}",
				InputParam: param,
				InputBody:  body,
			}
			req, err := NewRequest(opt)
			require.Emptyf(t, err, "case-%d", i)

			httpWriter := httptest.NewRecorder()
			restful.DefaultContainer.DoNotRecover(false)
			restful.DefaultContainer.Dispatch(httpWriter, req.Request)
			require.Equalf(t, 200, httpWriter.Code, "case-%d %s %s", i, req, httpWriter.Body.String())
		}
	}

}
