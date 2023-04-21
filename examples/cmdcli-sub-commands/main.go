package main

import (
	"context"
	"os"

	"github.com/spf13/cobra"
	"github.com/yubo/apiserver/components/version/verflag"
	"github.com/yubo/apiserver/pkg/proc"
	"k8s.io/klog/v2"
)

const (
	moduleName = "daemon-and-subcmd"
)

func main() {
	if err := newRootCmd().Execute(); err != nil {
		os.Exit(proc.PrintErrln(err))
	}
}

func newRootCmd() *cobra.Command {
	pc := proc.NewProcess()
	cmd := pc.NewRootCmd(
		proc.WithRun(root),
		proc.WithName(moduleName),
	)

	cmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		verflag.PrintAndExitIfRequested()
		return nil
	}

	cmd.AddCommand(newM1Cmd())
	return cmd
}

func root(ctx context.Context) error {
	klog.Infof("Press ctrl-c to leave the daemon process")
	return nil
}

// start with mainloop
func newM1Cmd() *cobra.Command {
	type config struct {
		M1 string `json:"m1" flag:"m1" default:"xxx" description:"m1 name"`
	}

	pc := proc.NewProcess()
	pc.AddConfig("m1", &config{}, proc.WithConfigGroup("m1"))

	run := func(ctx context.Context) error {
		cf := &config{}
		if err := pc.ReadConfig("m1", cf); err != nil {
			return err
		}

		klog.InfoS("m1", "name", cf.M1)
		return nil
	}

	return pc.NewCmd(proc.WithRun(run), proc.WithName("m1"), proc.WithoutLoop())
}
