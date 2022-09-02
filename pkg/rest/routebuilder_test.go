package rest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"testing"

	"github.com/emicklei/go-restful/v3"
	"github.com/stretchr/testify/require"
	"github.com/yubo/apiserver/pkg/client"
	"github.com/yubo/golib/util"
	"k8s.io/klog/v2"
)

func init() {
	klog.InitFlags(nil)
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
		container := NewBaseContainer()
		WsRouteBuild(&WsOption{
			Path:               "/dirs",
			GoRestfulContainer: container,
			Routes: []WsRoute{{
				Method: "POST", SubPath: "/{dir}/hosts/{name}",
				Handle: func(w http.ResponseWriter, req *http.Request, param *GetHost, body *GetHostBody) {
					require.Equalf(t, c.param, param, "case-%d", i)
					require.Equalf(t, c.body, body, "case-%d", i)
				}},
			},
		})

		// start server
		testServer := httptest.NewServer(http.Handler(container))
		defer testServer.Close()

		// start client
		cli, err := restClient(testServer)
		require.NoError(t, err)

		err = cli.Post().Prefix("/dirs/{dir}/hosts/{name}").
			Params(c.param).
			Body(c.body).Do(context.Background()).Error()
		require.NoErrorf(t, err, "case-%d", i)
	}
}

func TestRouteBuild(t *testing.T) {
	type createUserParam struct {
		Authorization string `param:"header" name:"Authorization"`
		Namespace     string `param:"path"`
	}
	type createUserRequest struct {
		Name     string `json:"name"`
		Age      int    `json:"age"`
		Nickname string `json:"nickname"`
	}

	type userParam struct {
		Name string `param:"path"`
	}

	type updateUserInput struct {
		Age      *int    `json:"age"`
		Nickname *string `json:"nickname"`
	}

	type User struct {
		Name     string `json:"name"`
		Age      int    `json:"age"`
		Nickname string `json:"nickname"`
	}

	cases := []struct {
		method string
		path   string
		param  interface{}
		body   interface{}
		resp   interface{}
		err    error
	}{{
		"POST",
		"/api/v1/users",
		nil,
		&createUserRequest{Name: "tom", Age: 16, Nickname: "zhangsan"},
		&User{Name: "tom", Age: 16, Nickname: "zhangsan"},
		nil,
	}, {
		"POST",
		"/api/v2/namespaces/{namespace}/users",
		&createUserParam{Authorization: "Basic 123", Namespace: "default"},
		&createUserRequest{Name: "tom", Age: 16, Nickname: "zhangsan"},
		&User{Name: "tom", Age: 16, Nickname: "zhangsan"},
		nil,
	}, {
		"GET",
		"/api/v1/users/{name}",
		&userParam{Name: "tom"},
		nil,
		&User{Name: "tom", Age: 16, Nickname: "zhangsan"},
		nil,
	}, {
		"PUT",
		"/api/v1/users/{name}",
		&userParam{Name: "tom"},
		&updateUserInput{Age: util.Int(18)},
		&User{Name: "tom", Age: 16, Nickname: "zhangsan"},
		nil,
	}, {
		"DELETE",
		"/api/v1/users/{name}",
		&userParam{Name: "tom"},
		nil,
		&User{Name: "tom", Age: 16, Nickname: "zhangsan"},
		nil,
	}}

	for _, c := range cases {
		t.Run(c.method+c.path, func(t *testing.T) {
			container := NewBaseContainer()

			WsRouteBuild(&WsOption{
				Path:               "/api",
				GoRestfulContainer: container,
				Routes: []WsRoute{{
					Method: "POST", SubPath: "/v1/users",
					Handle: func(w http.ResponseWriter, req *http.Request, body *createUserRequest) (*User, error) {
						require.Equal(t, c.body, body)
						return c.resp.(*User), c.err
					},
				}, {
					Method: "POST", SubPath: "/v2/namespaces/{namespace}/users",
					Handle: func(w http.ResponseWriter, req *http.Request, param *createUserParam, body *createUserRequest) (*User, error) {
						require.Equal(t, c.param, param)
						require.Equal(t, c.body, body)
						return c.resp.(*User), c.err
					},
				}, {
					Method: "GET", SubPath: "/v1/users/{name}",
					Handle: func(w http.ResponseWriter, req *http.Request, param *userParam) (*User, error) {
						require.Equal(t, c.param, param)
						return c.resp.(*User), c.err
					},
				}, {
					Method: "PUT", SubPath: "/v1/users/{name}",
					Handle: func(w http.ResponseWriter, req *http.Request, param *userParam, body *updateUserInput) (*User, error) {
						require.Equal(t, c.param, param)
						return c.resp.(*User), c.err
					},
				}, {
					Method: "DELETE", SubPath: "/v1/users/{name}",
					Handle: func(w http.ResponseWriter, req *http.Request, param *userParam) (*User, error) {
						require.Equal(t, c.param, param)
						return c.resp.(*User), c.err
					},
				}},
			})

			testServer := httptest.NewServer(http.Handler(container))
			defer testServer.Close()

			// start client
			cli, _ := restClient(testServer)

			req := cli.Verb(c.method).Prefix(c.path).Debug()

			if c.param != nil {
				req = req.Params(c.param)
			}
			if c.body != nil {
				req = req.Body(c.body)
			}

			result := req.Do(context.Background())

			if c.resp == nil {
				err := result.Error()
				require.Equal(t, c.err, err)
				return
			}

			resp := newInterfaceFromInterface(c.resp)
			err := result.Into(resp)
			require.Equal(t, c.err, err)
			require.Equal(t, c.resp, resp)

		})
	}
}

func dumpRequest(t *testing.T) func(*restful.Request, *restful.Response, *restful.FilterChain) {
	return func(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
		b, _ := httputil.DumpRequest(req.Request, true)
		t.Logf("%s", string(b))

		chain.ProcessFilter(req, resp)
	}
}

func restClient(testServer *httptest.Server) (*client.RESTClient, error) {
	return client.RESTClientFor(&client.Config{Host: testServer.URL})
}
