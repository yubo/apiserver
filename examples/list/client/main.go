package main

import (
	"context"
	"os"

	"github.com/yubo/apiserver/pkg/client"
	"github.com/yubo/apiserver/pkg/proc"
	"github.com/yubo/golib/api"
)

type User struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type ListInput struct {
	api.PageParams
}

type ListOutput struct {
	Total int     `json:"total"`
	List  []*User `json:"list"`
}

func run() error {
	req, err := client.NewRequest("127.0.0.1:8080",
		client.WithParams(&ListInput{
			PageParams: api.PageParams{
				PageSize: 10,
			},
		}),
		client.WithPath("/users"),
		client.WithOutput(&ListOutput{}),
	)
	if err != nil {
		return err
	}
	return req.Pager(os.Stdout, false).Do(context.Background())
}

func main() {
	os.Exit(proc.PrintErrln(run()))
}
