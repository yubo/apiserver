package main

import (
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/yubo/golib/cli"
	"github.com/yubo/golib/proc"

	_ "github.com/yubo/apiserver/pkg/server/register"
)

const (
	moduleName = "prometheus-metrics.examples"
)

var (
	opsProcessed = promauto.NewCounter(prometheus.CounterOpts{
		Name: "myapp_processed_ops_total",
		Help: "The total number of processed events",
	})
)

func recordMetrics() {
	go func() {
		for {
			opsProcessed.Inc()
			time.Sleep(2 * time.Second)
		}
	}()
}

func main() {
	recordMetrics()

	code := cli.Run(proc.NewRootCmd())
	os.Exit(code)
}
