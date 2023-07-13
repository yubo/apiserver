/*
Copyright 2017 The Kubernetes Authors.

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

package options

import (
	"github.com/yubo/apiserver/pkg/server"
	"github.com/yubo/golib/runtime/serializer"
)

type FeatureOptions struct {
	EnableProfiling           bool   `json:"enableProfiling" flag:"profiling" description:"Enable profiling via web interface host:port/debug/pprof/"`
	DebugSocketPath           string `json:"debugSocketPath" flag:"debug-socket-path" description:"Use an unprotected (no authn/authz) unix-domain socket for profiling with the given path"`
	EnableContentionProfiling bool   `json:"enableContentionProfiling" flag:"contention-profiling" description:"Enable block profiling, if profiling is enabled"`
	EnableIndex               bool   `json:"enableIndex"`
	EnableMetrics             bool   `json:"enableMetrics"`
	SkipOpenAPIInstallation   bool   `json:"skipOpenAPIInstallation" description:"enable OpenAPI/Swagger"`
	EnableExpvar              bool   `json:"enableExpvar"`
	EnableHealthz             bool   `json:"enableHealthz"`
}

func NewFeatureOptions() *FeatureOptions {
	defaults := server.NewConfig(serializer.CodecFactory{})

	return &FeatureOptions{
		EnableProfiling:           defaults.EnableProfiling,
		DebugSocketPath:           defaults.DebugSocketPath,
		EnableContentionProfiling: defaults.EnableContentionProfiling,
		EnableIndex:               defaults.EnableIndex,
		EnableMetrics:             defaults.EnableMetrics,
		SkipOpenAPIInstallation:   defaults.SkipOpenAPIInstallation,
		EnableExpvar:              defaults.EnableExpvar,
		EnableHealthz:             defaults.EnableHealthz,
	}
}

func (o *FeatureOptions) ApplyTo(c *server.Config) error {
	if o == nil {
		return nil
	}

	c.EnableProfiling = o.EnableProfiling
	c.DebugSocketPath = o.DebugSocketPath
	c.EnableContentionProfiling = o.EnableContentionProfiling
	c.EnableIndex = o.EnableIndex
	c.EnableMetrics = o.EnableMetrics
	c.SkipOpenAPIInstallation = o.SkipOpenAPIInstallation
	c.EnableExpvar = o.EnableExpvar
	c.EnableHealthz = o.EnableHealthz

	return nil
}

func (o *FeatureOptions) Validate() []error {
	if o == nil {
		return nil
	}

	errs := []error{}
	return errs
}
