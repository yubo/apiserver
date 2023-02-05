package proc

import (
	"reflect"

	"github.com/spf13/pflag"
	"github.com/yubo/golib/configer"
	"k8s.io/klog/v2"
)

func (p *Process) BindRegisteredFlags(fs *pflag.FlagSet) error {
	for _, v := range p.configs {
		var fs_ *pflag.FlagSet
		if v.group != "" {
			fs_ = p.namedFlagSets.FlagSet(v.group)
		} else if v.fs != nil {
			fs_ = v.fs
		} else {
			fs_ = fs
		}

		klog.V(10).InfoS("configVar", "path", v.path, "group", v.group, "type", reflect.TypeOf(v.sample).String())
		if err := p.ConfigVar(fs_, v.path, v.sample, v.opts...); err != nil {
			return err
		}
	}

	// for namedFlagSets
	for _, f := range p.namedFlagSets.FlagSets {
		fs.AddFlagSet(f)
	}

	return nil
}

// ConfigVar: set config fields to yaml configfile reader and pflags.FlagSet from sample
func (p *Process) ConfigVar(fs *pflag.FlagSet, path string, sample interface{}, opts ...configer.ConfigFieldsOption) error {
	return p.configer.Var(fs, path, sample, opts...)
}

func (p *Process) Configer() configer.ParsedConfiger {
	return p.parsedConfiger
}

func (p *Process) ReadConfig(path string, into interface{}) error {
	return p.parsedConfiger.Read(path, into)
}

// deprecated: use configOptions instead of it
//type ConfigOps struct {
//	group  string
//	fs     *pflag.FlagSet
//	path   string
//	sample interface{}
//	opts   []configer.ConfigFieldsOption
//}

func (p *Process) AddConfig(path string, sample interface{}, opts ...ConfigOption) error {
	cf := &configOptions{
		path:   path,
		sample: sample,
	}

	for _, opt := range opts {
		opt(cf)
	}

	if tagsGetter, ok := sample.(tagsGetter); ok {
		cf.opts = append(cf.opts, configer.WithTags(tagsGetter.GetTags))
	}

	p.configs = append(p.configs, cf)

	return nil
}

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
