package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/yubo/apiserver/components/cli"
	"github.com/yubo/apiserver/pkg/proc"
	"github.com/yubo/golib/util/yaml"
)

func main() {
	// register module config
	proc.AddConfig("example", newConfig(), proc.WithConfigGroup("example"))

	code := cli.Run(proc.NewRootCmd(proc.WithRun(start), proc.WithName("logger"), proc.WithoutLoop()))
	os.Exit(code)
}

type config struct {
	UserName string `json:"userName" flag:"user-name" env:"USER_NAME" description:"user name"`
	UserAge  int    `json:"userAge" flag:"user-age" env:"USER_AGE" description:"user age"`
	City     string `json:"city" flag:"city" env:"USER_CITY" default:"beijing" description:"city"`
}

func newConfig() *config {
	return &config{UserName: "Anonymous"}
}

func start(ctx context.Context) error {
	cf := newConfig()
	if err := proc.ReadConfig("example", cf); err != nil {
		return err
	}

	b, _ := yaml.Marshal(cf)
	fmt.Printf("%s\n%s\n%s\n", strings.Repeat("=", 37), string(b), strings.Repeat("=", 37))

	return nil
}
