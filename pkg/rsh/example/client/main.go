package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/yubo/apiserver/pkg/rest"
	"github.com/yubo/apiserver/pkg/rsh"
	"github.com/yubo/golib/util"
	"k8s.io/klog/v2"
)

type ExecOption struct {
	Cmd []string `param:"query" description:"cmd desc"`
	Foo string   `param:"query" description:"foo desc"`
	Bar int      `param:"query" description:"bar desc"`
}

func main() {

	klog.InitFlags(nil)
	flag.Set("v", "3")
	flag.Parse()

	opt := &rest.RequestOptions{
		Url:    "http://localhost:18080/exec",
		Bearer: util.String("1"),
		InputParam: &ExecOption{
			Cmd: os.Args[1:],
			Foo: os.Args[0],
			Bar: len(os.Args),
		},
		InputBody: "hello, world",
	}

	if err := rsh.RshRequest(opt); err != nil {
		fmt.Printf("[err]Communication error: %v\n", err)
	}
	os.Exit(0)

}
