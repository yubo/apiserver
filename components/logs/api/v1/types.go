/*
Copyright 2021 The Kubernetes Authors.

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

package v1

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/yubo/golib/api"
	"github.com/yubo/golib/api/resource"
	"github.com/yubo/golib/configer"
)

// Supported output formats.
const (
	// DefaultLogFormat is the traditional klog output format.
	DefaultLogFormat = "text"

	// JSONLogFormat emits each log message as a JSON struct.
	JSONLogFormat = "json"
)

// The alpha or beta level of structs is the highest stability level of any field
// inside it. Feature gates will get checked during LoggingConfiguration.ValidateAndApply.

// LoggingConfiguration contains logging options.
type LoggingConfiguration struct {
	// Format Flag specifies the structure of log messages.
	// default value of format is `text`
	Format string `json:"format,omitempty" flag:"logging-format"`
	// Maximum number of nanoseconds (i.e. 1s = 1000000000) between log
	// flushes. Ignored if the selected logging backend writes log
	// messages without buffering.
	FlushFrequency api.Duration `json:"flushFrequency" flag:"log-flush-frequency"`
	// Verbosity is the threshold that determines which log messages are
	// logged. Default is zero which logs only the most important
	// messages. Higher values enable additional messages. Error messages
	// are always logged.
	Verbosity VerbosityLevel `json:"verbosity" flag:"v,v" description:"number for the log level verbosity"`
	// VModule overrides the verbosity threshold for individual files.
	// Only supported for "text" log format.
	VModule VModuleConfiguration `json:"vmodule,omitempty" flag:"vmodule" description:"comma-separated list of pattern=N settings for file-filtered logging (only works for text log format)"`
	// [Alpha] Options holds additional parameters that are specific
	// to the different logging formats. Only the options for the selected
	// format get used, but all of them get validated.
	// Only available when the LoggingAlphaOptions feature gate is enabled.
	Options FormatOptions `json:"options,omitempty"`
}

func (p *LoggingConfiguration) GetTags() map[string]*configer.FieldTag {
	formats := logRegistry.list()
	return map[string]*configer.FieldTag{
		"format": {Description: fmt.Sprintf("Sets the log format. Permitted formats: %s.", formats)},
	}
}

// FormatOptions contains options for the different logging formats.
type FormatOptions struct {
	// [Alpha] JSON contains options for logging format "json".
	// Only available when the LoggingAlphaOptions feature gate is enabled.
	JSON JSONOptions `json:"json,omitempty"`
}

// JSONOptions contains options for logging format "json".
type JSONOptions struct {
	// [Alpha] SplitStream redirects error messages to stderr while
	// info messages go to stdout, with buffering. The default is to write
	// both to stdout, without buffering. Only available when
	// the LoggingAlphaOptions feature gate is enabled.
	SplitStream bool `json:"splitStream,omitempty" flag:"log-json-split-stream" description:"[Alpha] In JSON format, write error messages to stderr and info messages to stdout. The default is to write a single stream to stdout. Enable the LoggingAlphaOptions feature gate to use this."`
	// [Alpha] InfoBufferSize sets the size of the info stream when
	// using split streams. The default is zero, which disables buffering.
	// Only available when the LoggingAlphaOptions feature gate is enabled.
	InfoBufferSize resource.QuantityValue `json:"infoBufferSize,omitempty" flag:"log-json-info-buffer-size" description:"[Alpha] In JSON format with split output streams, the info messages can be buffered for a while to increase performance. The default value of zero bytes disables buffering. The size can be specified as number of bytes (512), multiples of 1000 (1K), multiples of 1024 (2Ki), or powers of those (3M, 4G, 5Mi, 6Gi). Enable the LoggingAlphaOptions feature gate to use this."`
}

// VModuleConfiguration is a collection of individual file names or patterns
// and the corresponding verbosity threshold.
type VModuleConfiguration []VModuleItem

// VModuleItem defines verbosity for one or more files which match a certain
// glob pattern.
type VModuleItem struct {
	// FilePattern is a base file name (i.e. minus the ".go" suffix and
	// directory) or a "glob" pattern for such a name. It must not contain
	// comma and equal signs because those are separators for the
	// corresponding klog command line argument.
	FilePattern string `json:"filePattern"`
	// Verbosity is the threshold for log messages emitted inside files
	// that match the pattern.
	Verbosity VerbosityLevel `json:"verbosity"`
}

// VerbosityLevel represents a klog or logr verbosity threshold.
type VerbosityLevel uint32

func (v VerbosityLevel) String() string {
	return strconv.FormatUint(uint64(uint32(v)), 10)
}

func (v *VerbosityLevel) Set(s string) error {
	i, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return err
	}
	*v = VerbosityLevel(uint32(i))
	return nil
}

func (v VerbosityLevel) Type() string {
	return "VerbosityLevel"
}

// String returns the -vmodule parameter (comma-separated list of pattern=N).
func (v VModuleConfiguration) String() string {
	var patterns []string
	for _, item := range v {
		patterns = append(patterns, fmt.Sprintf("%s=%d", item.FilePattern, item.Verbosity))
	}
	return strings.Join(patterns, ",")
}

// Set parses the -vmodule parameter (comma-separated list of pattern=N).
func (v VModuleConfiguration) Set(value string) error {
	// This code mirrors https://github.com/kubernetes/klog/blob/9ad246211af1ed84621ee94a26fcce0038b69cd1/klog.go#L287-L313

	for _, pat := range strings.Split(value, ",") {
		if len(pat) == 0 {
			// Empty strings such as from a trailing comma can be ignored.
			continue
		}
		patLev := strings.Split(pat, "=")
		if len(patLev) != 2 || len(patLev[0]) == 0 || len(patLev[1]) == 0 {
			return fmt.Errorf("%q does not have the pattern=N format", pat)
		}
		pattern := patLev[0]
		// 31 instead of 32 to ensure that it also fits into int32.
		v2, err := strconv.ParseUint(patLev[1], 10, 31)
		if err != nil {
			return fmt.Errorf("parsing verbosity in %q: %v", pat, err)
		}
		v = append(v, VModuleItem{FilePattern: pattern, Verbosity: VerbosityLevel(v2)})
	}
	return nil
}

func (v VModuleConfiguration) Type() string {
	return "pattern=N,..."
}
