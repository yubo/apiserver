package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/yubo/golib/configer"
	"github.com/yubo/golib/proc"
	"sigs.k8s.io/yaml"
)

type config struct {
	UserName string `json:"userName" flag:"user-name" env:"USER_NAME" description:"user name"`
	UserAge  int    `json:"userAge" flag:"user-age" env:"USER_AGE" description:"user age"`
	City     string `json:"city" flag:"city" env:"USER_CITY" default:"beijing" description:"city"`
	License  string `json:"license" flag:"license" description:"license"`
}

func newConfig() *config {
	return &config{UserName: "Anonymous"}
}

const (
	moduleName = "example"
)

var (
	hookOps = []proc.HookOps{{
		Hook:     start,
		Owner:    moduleName,
		HookNum:  proc.ACTION_START,
		Priority: proc.PRI_MODULE,
	}}
)

func main() {
	if err := proc.NewRootCmd(proc.WithoutLoop()).Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func start(ctx context.Context) error {
	c := configer.ConfigerMustFrom(ctx)

	cf := newConfig()
	if err := c.Read(moduleName, cf); err != nil {
		return err
	}

	b, _ := yaml.Marshal(cf)
	fmt.Printf("%s\n%s\n%s\n", strings.Repeat("=", 37), string(b), strings.Repeat("=", 37))

	return nil
}

func init() {
	// register hookOps as a module
	proc.RegisterHooks(hookOps)

	// register config{} to configer.Factory
	proc.AddConfig(moduleName, newConfig(), proc.WithConfigGroup("example"))
}
