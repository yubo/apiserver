package config

type Config struct {
	Debug bool
}

func New() *Config {
	return &Config{}
}
