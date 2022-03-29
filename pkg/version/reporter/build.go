package reporter

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/spf13/cobra"
	"github.com/yubo/apiserver/pkg/version"
)

var (
	errAlreadyStarted = errors.New("reporter already started")
	errNotStarted     = errors.New("reporter not started")
)

type buildReporter struct {
	sync.Mutex

	buildTime time.Time
	active    bool
	closeCh   chan struct{}
	doneCh    chan struct{}
}

func (b *buildReporter) Start() error {
	const (
		base    = 10
		bitSize = 64
	)

	buildTime, err := time.Parse(time.RFC3339, version.Get().BuildDate)
	if err != nil {
		return err
	}

	b.Lock()
	defer b.Unlock()
	if b.active {
		return errAlreadyStarted
	}
	b.buildTime = buildTime
	b.active = true
	b.closeCh = make(chan struct{})
	b.doneCh = make(chan struct{})
	go b.report()
	return nil
}

func (b *buildReporter) report() {
	ver := version.Get()
	tags := map[string]string{
		"version":        ver.GitVersion,
		"commit":         ver.GitCommit,
		"build_date":     ver.BuildDate,
		"git_tree_state": ver.GitTreeState,
		"go_version":     ver.GoVersion,
	}

	buildInfoGauge := promauto.NewGauge(prometheus.GaugeOpts{
		Name:        "build_information",
		ConstLabels: tags,
	})

	buildAgeGauge := promauto.NewGauge(prometheus.GaugeOpts{
		Name:        "build_age",
		ConstLabels: tags,
	})

	buildInfoGauge.Set(1.0)
	buildAgeGauge.Set(float64(time.Since(b.buildTime)))

	ticker := time.NewTicker(time.Second * 10)
	defer func() {
		close(b.doneCh)
		ticker.Stop()
	}()

	for {
		select {
		case <-ticker.C:
			buildInfoGauge.Set(1.0)
			buildAgeGauge.Set(float64(time.Since(b.buildTime)))

		case <-b.closeCh:
			return
		}
	}
}

func (b *buildReporter) Stop() error {
	b.Lock()
	defer b.Unlock()
	if !b.active {
		return errNotStarted
	}
	close(b.closeCh)
	<-b.doneCh
	b.active = false
	return nil
}

func NewVersionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "show version, git commit",
		RunE: func(cmd *cobra.Command, args []string) error {
			ver := version.Get()
			//fmt.Printf("Major:           %s\n", ver.Major)
			//fmt.Printf("Minor:           %s\n", ver.Minor)
			fmt.Printf("Git Version:     %s\n", ver.GitVersion)
			fmt.Printf("Git Commit:      %s\n", ver.GitCommit)
			fmt.Printf("Git Tree State:  %s\n", ver.GitTreeState)
			fmt.Printf("Build Date:      %s\n", ver.BuildDate)
			fmt.Printf("Go Version:      %s\n", ver.GoVersion)
			fmt.Printf("Compiler:        %s\n", ver.Compiler)
			fmt.Printf("Platform:        %s\n", ver.Platform)
			return nil
		},
	}
	return cmd
}
