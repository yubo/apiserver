package main

import (
	"context"
	"os"
	"time"

	"github.com/yubo/apiserver/components/cli"
	"github.com/yubo/apiserver/components/metrics"
	"github.com/yubo/apiserver/components/metrics/legacyregistry"
	"github.com/yubo/apiserver/pkg/proc"

	_ "github.com/yubo/apiserver/pkg/server/register"
)

var (
	opsProcessed = metrics.NewCounter(&metrics.CounterOpts{
		Name: "myapp_processed_ops_total",
		Help: "The total number of processed events",
	})
)

func main() {
	legacyregistry.MustRegister(opsProcessed)
	os.Exit(cli.Run(proc.NewRootCmd(proc.WithRun(start))))
}

func start(ctx context.Context) error {
	go func() {
		for {
			opsProcessed.Inc()
			time.Sleep(2 * time.Second)
		}
	}()

	return nil
}
