package main

import (
	"context"
	"examples/gen-sdk/client/openapi"
	"fmt"
	"os"
)

func newCli() *openapi.APIClient {
	config := &openapi.Configuration{
		DefaultHeader: make(map[string]string),
		UserAgent:     "OpenAPI-Generator/1.0.0/go",
		Host:          "localhost:8080",
		Scheme:        "http",
		Debug:         false,
		Servers: openapi.ServerConfigurations{
			{
				URL:         "",
				Description: "No description provided",
			},
		},
		OperationServers: map[string]openapi.ServerConfigurations{},
	}
	return openapi.NewAPIClient(config)
}

func main() {
	cli := newCli()
	userName := "Hamilton"

	// create user
	{
		ham := *openapi.NewMainCreateUserInput(userName, "Ham", "0086-123456")

		resp, r, err := cli.UserApi.CreateUser(context.Background()).Body(ham).Execute()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error when calling `UserApi.CreateUser``: %v\n", err)
			fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
			return
		}
		fmt.Fprintf(os.Stdout, "Response from `UserApi.CreateUser`: %v\n", resp)
		// output:
		// Response from `UserApi.CreateUser`: {Hamilton Ham 0086-123456}
	}

	// get user
	{
		resp, r, err := cli.UserApi.GetUser(context.Background(), userName).Execute()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error when calling `UserApi.GetUser``: %v\n", err)
			fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
			return
		}
		fmt.Fprintf(os.Stdout, "Response from `UserApi.GetUser`: %v\n", resp)
		// output:
		// Response from `UserApi.GetUser`: {Hamilton Ham 0086-123456}

	}

	// update user
	{
		body := *openapi.NewMainUpdateUserBody("Ham", "0086-888888")
		resp, r, err := cli.UserApi.UpdateUser(context.Background(), userName).Body(body).Execute()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error when calling `UserApi.UpdateUser``: %v\n", err)
			fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
			return
		}
		fmt.Fprintf(os.Stdout, "Response from `UserApi.UpdateUser`: %v\n", resp)
		// output:
		// Response from `UserApi.UpdateUser`: {Hamilton Ham 0086-888888}
	}

	// list user
	{
		resp, r, err := cli.UserApi.GetUsers(context.Background()).Count(true).PageSize(10).CurrentPage(0).Sorter("name").Order("asc").Execute()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error when calling `UserApi.GetUsers``: %v\n", err)
			fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
			return
		}
		fmt.Fprintf(os.Stdout, "Response from `UserApi.GetUsers`: %v\n", resp)
		// output:
		// Response from `UserApi.GetUsers`: {[{Hamilton Ham 0086-888888}] 1}
	}

	// delete user
	{
		resp, r, err := cli.UserApi.DeleteUser(context.Background(), userName).Execute()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error when calling `UserApi.DeleteUser``: %v\n", err)
			fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
			return
		}
		fmt.Fprintf(os.Stdout, "Response from `UserApi.DeleteUser`: %v\n", resp)
		// output:
		// Response from `UserApi.DeleteUser`: {Hamilton Ham 0086-888888}
	}
}
