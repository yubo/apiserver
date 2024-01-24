package proc

import (
	"github.com/spf13/pflag"
	"github.com/yubo/golib/configer"
)

type tagsGetter interface {
	GetTags() map[string]*configer.FieldTag
}

type configOptions struct {
	// component's path/name
	path string

	// for flags group
	group string

	// config sample struct
	sample interface{}

	// fs.AddFlagSet() will be called by BindRegisteredFlags
	fs *pflag.FlagSet

	opts []configer.ConfigFieldsOption
}

type ConfigOption func(*configOptions)

func WithConfigGroup(group string) ConfigOption {
	return func(o *configOptions) {
		o.group = group
	}
}

func WithConfigFlagSet(fs *pflag.FlagSet) ConfigOption {
	return func(o *configOptions) {
		o.fs = fs
	}
}
