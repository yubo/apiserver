package main

import (
	"context"
	"os"

	"github.com/yubo/apiserver/pkg/cmdcli"
	"github.com/yubo/apiserver/pkg/rest"
	"k8s.io/klog/v2"
)

type User struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type ListInput struct {
	rest.PageParams
}

type ListOutput struct {
	Total int     `json:"total"`
	List  []*User `json:"list"`
}

func run() error {
	req, err := cmdcli.NewRequest("127.0.0.1:8080",
		cmdcli.WithParams(&ListInput{
			PageParams: rest.PageParams{
				PageSize: 10,
			},
		}),
		cmdcli.WithPath("/users"),
		cmdcli.WithOutput(&ListOutput{}),
	)
	if err != nil {
		return err
	}
	return req.Pager(os.Stdout, false).Do(context.Background())
}

func main() {
	if err := run(); err != nil {
		klog.Error(err)
		os.Exit(1)
	}
}
