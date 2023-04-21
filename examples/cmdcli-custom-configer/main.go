package main

import (
	"context"
	"math/rand"
	"os"
	"runtime"
	"time"

	"github.com/spf13/cobra"
	"github.com/yubo/apiserver/components/version/verflag"
	"github.com/yubo/apiserver/pkg/proc"
	"k8s.io/klog/v2"
)

const (
	moduleName = "custom-configer-cmds"
)

func main() {
	if err := newRootCmd().Execute(); err != nil {
		os.Exit(proc.PrintErrln(err))
	}
}

func newRootCmd() *cobra.Command {
	rand.Seed(time.Now().UnixNano())
	runtime.GOMAXPROCS(runtime.NumCPU())
	ctx := context.TODO()

	cmd := &cobra.Command{
		Use: moduleName,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			verflag.PrintAndExitIfRequested()
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			verflag.PrintAndExitIfRequested()
			return nil
		},
	}

	cmd.AddCommand(
		newM1Cmd(ctx), // loop mode
		newM2Cmd(ctx), // without loop
		newM3Cmd(ctx), // without proc
	)
	verflag.AddFlags(cmd.PersistentFlags())

	return cmd
}

// start with mainloop
func newM1Cmd(ctx context.Context) *cobra.Command {
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
		klog.Infof("Press ctrl-c to leave the daemon process")
		return nil
	}

	return pc.NewCmd(proc.WithRun(run), proc.WithName("m1"))
}

// without loop
func newM2Cmd(ctx context.Context) *cobra.Command {
	type config struct {
		M2 string `json:"m2" flag:"m2" default:"yyy" description:"m2 name"`
	}

	pc := proc.NewProcess()
	pc.AddConfig("m2", &config{}, proc.WithConfigGroup("m2"))

	run := func(ctx context.Context) error {
		cf := &config{}
		if err := pc.ReadConfig("m2", cf); err != nil {
			return err
		}

		klog.InfoS("m2", "name", cf.M2)
		return nil
	}
	return pc.NewCmd(proc.WithRun(run), proc.WithName("m2"), proc.WithoutLoop())
}

// custome cmd
func newM3Cmd(ctx context.Context) *cobra.Command {
	var name string

	cmd := &cobra.Command{
		Use:          "m3",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			klog.InfoS("m3", "name", name)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "zzz", "m3 name")
	return cmd
}
