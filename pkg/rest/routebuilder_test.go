package rest

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/emicklei/go-restful/v3"
	"github.com/stretchr/testify/assert"
	"github.com/yubo/apiserver/pkg/client"
	"github.com/yubo/golib/util"
	"k8s.io/klog/v2"
)

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

	codec := ParameterCodec
	for i, c := range cases {
		container := restful.NewContainer()
		ws := new(restful.WebService).Path("").Consumes(MIME_JSON)
		ws.Route(ws.POST("/dirs/{dir}/hosts/{name}").
			Consumes(MIME_JSON).
			Produces(MIME_JSON).
			Filter(dbgFilter).
			To(func(req *restful.Request, resp *restful.Response) {
				param := GetHost{}
				body := GetHostBody{}
				err := readEntity(req, &param, &body, codec)
				assert.Emptyf(t, err, "case-%d", i)
				assert.Equalf(t, c.param, &param, "case-%d", i)
				assert.Equalf(t, c.body, &body, "case-%d", i)
			}))
		container.Add(ws)

		// start server
		testServer := httptest.NewServer(http.Handler(container))
		defer testServer.Close()

		// start client
		cli, err := restClient(testServer)
		assert.NoError(t, err)

		err = cli.Post().Prefix("/dirs/{dir}/hosts/{name}").
			VersionedParams(c.param, codec).
			Body(c.body).Do(context.Background()).Error()
		assert.NoErrorf(t, err, "case-%d", i)
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

	cases := []interface{}{}
	cases = append(cases,
		&UpdateInput{Namespace: "default", Name: "tom"},
		&UpdateUserInput{Age: util.Int(26), DisplayName: util.String("sonic2020")},
	)

	codec := ParameterCodec

	for i := 0; i < len(cases)/2; i++ {
		container := restful.NewContainer()
		param := cases[2*i]
		body := cases[2*i+1]
		ws := new(restful.WebService).Path("").Consumes(MIME_JSON)
		WsRouteBuild(&WsOption{
			Ws: ws.Path("").Produces(MIME_JSON).Consumes(MIME_JSON),
			Routes: []WsRoute{{
				Method: "PUT", SubPath: "/namespaces/{namespace}/users/{name}",
				Desc: "update user",
				Handle: func(w http.ResponseWriter, req *http.Request, p *UpdateInput, b *UpdateUserInput) {
					assert.Equalf(t, param, p, "case-%d", i)
					assert.Equalf(t, body, b, "case-%d", i)
					return
				}},
			},
		})
		container.Add(ws)
		testServer := httptest.NewServer(http.Handler(container))
		defer testServer.Close()

		// start client
		c, err := restClient(testServer)
		assert.NoError(t, err)

		err = c.Put().Prefix("namespaces/{namespace}/users/{name}").
			VersionedParams(param, codec).
			Body(body).Do(context.Background()).Error()
		assert.NoError(t, err)
	}

}

func TestWsRouteBuildWithResponse(t *testing.T) {
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

	cases := []interface{}{}
	cases = append(cases,
		&UpdateInput{Namespace: "default", Name: "tom"},
		&UpdateUserInput{Age: util.Int(26), DisplayName: util.String("sonic2020")},
		&User{Name: util.String("tom"), Age: util.Int(26), DisplayName: util.String("sonic2020")},
	)

	codec := ParameterCodec

	for i := 0; i < len(cases)/3; i++ {
		container := restful.NewContainer()
		param := cases[3*i]
		body := cases[3*i+1]
		user := cases[3*i+2].(*User)
		ws := new(restful.WebService).Path("").Consumes(MIME_JSON)
		WsRouteBuild(&WsOption{
			Ws: ws.Path("").Produces(MIME_JSON).Consumes(MIME_JSON),
			Routes: []WsRoute{{
				Method: "PUT", SubPath: "/namespaces/{namespace}/users/{name}",
				Desc: "update user",
				Handle: func(w http.ResponseWriter, req *http.Request, p *UpdateInput, b *UpdateUserInput) (*User, error) {
					assert.Equalf(t, param, p, "case-%d", i)
					assert.Equalf(t, body, b, "case-%d", i)
					return user, nil
				}},
			},
		})
		container.Add(ws)
		testServer := httptest.NewServer(http.Handler(container))
		defer testServer.Close()

		// start client
		c, _ := restClient(testServer)

		got := &User{}
		err := c.Put().Prefix("namespaces/{namespace}/users/{name}").
			VersionedParams(param, codec).
			Body(body).Do(context.Background()).Into(got)
		assert.NoError(t, err)
		assert.Equalf(t, user, got, "get user from api")
	}

}

func restClient(testServer *httptest.Server) (*client.RESTClient, error) {
	c, err := client.RESTClientFor(&client.Config{
		Host:     testServer.URL,
		Username: "user",
		Password: "pass",
	})
	return c, err
}
