package kv

import (
	"context"
	"fmt"

	"github.com/yubo/golib/proc"
	"github.com/yubo/golib/util"
	"go.uber.org/zap"
)

const (
	moduleName = "kv"
)

// Configuration defines configuration for logging.
type Config struct {
	File   string                 `json:"file" yaml:"file"`
	Level  string                 `json:"level" yaml:"level"`
	Fields map[string]interface{} `json:"fields" yaml:"fields"`
}

// BuildLogger builds a new Logger based on the configuration.
func (cfg Config) BuildLogger() (*zap.Logger, error) {
	zc := zap.Config{
		Level:             zap.NewAtomicLevelAt(zap.InfoLevel),
		Development:       false,
		DisableCaller:     true,
		DisableStacktrace: true,
		Sampling: &zap.SamplingConfig{
			Initial:    100,
			Thereafter: 100,
		},
		Encoding:         "json",
		EncoderConfig:    zap.NewProductionEncoderConfig(),
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stdout"},
		InitialFields:    cfg.Fields,
	}

	if cfg.File != "" {
		zc.OutputPaths = append(zc.OutputPaths, cfg.File)
		zc.ErrorOutputPaths = append(zc.ErrorOutputPaths, cfg.File)
	}

	if len(cfg.Level) != 0 {
		var parsedLevel zap.AtomicLevel
		if err := parsedLevel.UnmarshalText([]byte(cfg.Level)); err != nil {
			return nil, fmt.Errorf("unable to parse log level %s: %v", cfg.Level, err)
		}
		zc.Level = parsedLevel
	}

	return zc.Build()
}

func (p Config) String() string {
	return util.Prettify(p)
}

func (p *Config) Validate() error {
	if p.Level == "" {
		p.Level = "info"
	}
	return nil
}

type Module struct {
	*Config
	name   string
	logger *zap.Logger
}

var (
	_module = &Module{name: moduleName}
	hookOps = []proc.HookOps{{
		Hook:     _module.start,
		Owner:    moduleName,
		HookNum:  proc.ACTION_START,
		Priority: proc.PRI_SYS_PRESTART,
	}}
)

// TODO
func (p *Module) start(ctx context.Context) error {
	c := proc.ConfigerMustFrom(ctx)
	cf := &Config{}
	if err := c.Read("kv", cf); err != nil {
		return err
	}
	p.Config = cf

	return nil
}

func init() {
	proc.RegisterHooks(hookOps)
}
