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

const (
	moduleName = "example"
)

var (
	_module = &module{name: moduleName}
)

type config struct {
	UserName string `json:"userName" flag:"user-name" env:"USER_NAME" description:"user name"`
	UserAge  int    `json:"userAge" flag:"user-age" env:"USER_AGE" description:"user age"`
	City     string `json:"city" flag:"city" env:"USER_CITY" default:"beijing" description:"city"`
}

type module struct {
	name string
}

func newConfig() *config {
	return &config{UserName: "Anonymous"}
}

func (p *module) start(ctx context.Context) error {
	cf := newConfig()
	if err := proc.ReadConfig(p.name, cf); err != nil {
		return err
	}

	b, _ := yaml.Marshal(cf)
	fmt.Printf("%s\n%s\n%s\n", strings.Repeat("=", 37), string(b), strings.Repeat("=", 37))

	return nil
}

func main() {
	// register module config
	proc.AddConfig(moduleName, newConfig(), proc.WithConfigGroup("example"))

	code := cli.Run(proc.NewRootCmd(
		proc.WithRun(_module.start),
		proc.WithName("logger"),
		proc.WithoutLoop(),
	))
	os.Exit(code)
}
