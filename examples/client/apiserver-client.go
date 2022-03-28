package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/yubo/apiserver/pkg/cmdcli"
	"github.com/yubo/apiserver/pkg/responsewriters"
	"github.com/yubo/apiserver/pkg/rest"
	"github.com/yubo/golib/scheme"
)

type User struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func requestWithRest(server *httptest.Server) error {
	user := User{}

	c, err := rest.RESTClientFor(&rest.Config{
		Host: server.URL,
		ContentConfig: rest.ContentConfig{
			NegotiatedSerializer: scheme.Codecs,
		},
	})
	if err != nil {
		return err
	}

	if err := c.Get().Prefix("/").Do(context.Background()).Into(&user); err != nil {
		return err
	}

	fmt.Printf("resp1: %+v\n", user)
	return nil
}

func requestWithCmdcli(server *httptest.Server) error {
	user := User{}

	req, err := cmdcli.NewRequest(server.URL,
		cmdcli.WithOutput(&user),
		cmdcli.WithMethod("GET"),
		cmdcli.WithPrefix("/"),
	)
	if err != nil {
		return err
	}

	if err := req.Do(context.Background()); err != nil {
		return err
	}

	fmt.Printf("resp2: %+v\n", user)
	return nil
}

func main() {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, req *http.Request) {
			responsewriters.WriteRawJSON(200, &User{Name: "tom", Age: 14}, w)
		}))
	defer server.Close()

	if err := requestWithRest(server); err != nil {
		fmt.Println("err:", err)
	}
	if err := requestWithCmdcli(server); err != nil {
		fmt.Println("err:", err)
	}
	// output:
	// resp1: {Name:tom Age:14}
	// resp2: {Name:tom Age:14}

}
