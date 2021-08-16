package main

import (
	"os"

	"github.com/yubo/apiserver/pkg/cmdcli"
	"github.com/yubo/apiserver/pkg/rest"
	"k8s.io/klog/v2"
)

type Item struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type ListInput struct {
	rest.Pagination
}

type ListOutput struct {
	Total int     `json:"total"`
	List  []*Item `json:"list"`
}

func run() error {
	req, err := cmdcli.NewRequest("127.0.0.1:18080",
		cmdcli.WithInput(&ListInput{
			Pagination: rest.Pagination{
				PageSize: 10,
			},
		}),
		cmdcli.WithOutput(&ListOutput{}),
	)
	if err != nil {
		return err
	}
	return req.Pager(os.Stdout, false).Run()
}

func main() {
	if err := run(); err != nil {
		klog.Error(err)
		os.Exit(1)
	}
}
