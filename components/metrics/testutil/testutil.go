/*
Copyright 2019 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package testutil

import (
	"fmt"
	"io"

	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/yubo/apiserver/components/metrics"
	"github.com/yubo/golib/version"
)

// CollectAndCompare registers the provided Collector with a newly created
// pedantic Registry. It then does the same as GatherAndCompare, gathering the
// metrics from the pedantic Registry.
func CollectAndCompare(c metrics.Collector, expected io.Reader, metricNames ...string) error {
	lintProblems, err := testutil.CollectAndLint(c, metricNames...)
	if err != nil {
		return err
	}
	if err := getLintError(lintProblems); err != nil {
		return err
	}

	return testutil.CollectAndCompare(c, expected, metricNames...)
}

// GatherAndCompare gathers all metrics from the provided Gatherer and compares
// it to an expected output read from the provided Reader in the Prometheus text
// exposition format. If any metricNames are provided, only metrics with those
// names are compared.
func GatherAndCompare(g metrics.Gatherer, expected io.Reader, metricNames ...string) error {
	lintProblems, err := testutil.GatherAndLint(g, metricNames...)
	if err != nil {
		return err
	}
	if err := getLintError(lintProblems); err != nil {
		return err
	}

	return testutil.GatherAndCompare(g, expected, metricNames...)
}

// CustomCollectAndCompare registers the provided StableCollector with a newly created
// registry. It then does the same as GatherAndCompare, gathering the
// metrics from the pedantic Registry.
func CustomCollectAndCompare(c metrics.StableCollector, expected io.Reader, metricNames ...string) error {
	registry := metrics.NewKubeRegistry()
	registry.CustomMustRegister(c)

	return GatherAndCompare(registry, expected, metricNames...)
}

// NewFakeKubeRegistry creates a fake `KubeRegistry` that takes the input version as `build in version`.
// It should only be used in testing scenario especially for the deprecated metrics.
// The input version format should be `major.minor.patch`, e.g. '1.18.0'.
func NewFakeKubeRegistry(ver string) metrics.KubeRegistry {
	backup := metrics.BuildVersion
	defer func() {
		metrics.BuildVersion = backup
	}()

	metrics.BuildVersion = func() version.Info {
		return version.Info{
			GitVersion: fmt.Sprintf("v%s-alpha+1.12345", ver),
		}
	}

	return metrics.NewKubeRegistry()
}
