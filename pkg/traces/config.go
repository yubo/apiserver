package traces

import (
	"os"
	"path/filepath"

	"github.com/yubo/golib/util"
)

type Config struct {
	ServiceName       string            `yaml:"serviceName"`
	ContextHeaderName string            `yaml:"contextHeadername"`
	Attributes        map[string]string `yaml:"attributes"`

	OTel   *OTelConfig   `yaml:"otel"`
	Jaeger *JaegerConfig `yaml:"jaeger"`
}

type OTelConfig struct {
	Endpoint string `yaml:"endpoint"`
	Insecure bool   `yaml:"insecure"`
}

type JaegerConfig struct {
	Endpoint string `yaml:"endpoint"`
	Insecure bool   `yaml:"insecure"`
}

func newConfig() *Config {
	return &Config{
		ServiceName:       filepath.Base(os.Args[0]),
		ContextHeaderName: "",
	}
}

func (p Config) String() string {
	return util.Prettify(p)
}

func (p *Config) Validate() error {
	return nil
}
