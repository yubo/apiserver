package main

import (
	"context"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/yubo/apiserver/components/cli"
	"github.com/yubo/apiserver/pkg/proc"

	_ "github.com/yubo/apiserver/pkg/server/register"
)

var (
	opsProcessed = promauto.NewCounter(prometheus.CounterOpts{
		Name: "myapp_processed_ops_total",
		Help: "The total number of processed events",
	})
)

func main() {
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
