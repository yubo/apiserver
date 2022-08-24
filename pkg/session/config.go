package session

import (
	"fmt"

	"github.com/yubo/golib/api"
)

const (
	DefCookieName = "sid"
)

func NewConfig() *Config {
	return &Config{
		CookieName:     DefCookieName,
		SidLength:      32,
		HttpOnly:       true,
		GcInterval:     api.NewDuration("600s"),
		CookieLifetime: api.NewDuration("16h"),
		MaxIdleTime:    api.NewDuration("1h"),
		TableName:      "session",
	}
}

type Config struct {
	CookieName     string       `json:"cookieName"`
	SidLength      int          `json:"sidLength"`
	HttpOnly       bool         `json:"httpOnly"`
	Domain         string       `json:"domain"`
	GcInterval     api.Duration `json:"gcInterval"`
	CookieLifetime api.Duration `json:"cookieLifetime"`
	MaxIdleTime    api.Duration `json:"maxIdleTime"`
	TableName      string       `json:"tableName"`
}

func (p *Config) Validate() error {
	if p == nil {
		return nil
	}

	if p.SidLength <= 0 {
		return fmt.Errorf("invalid sid length %d", p.SidLength)
	}

	if p.CookieName == "" {
		p.CookieName = DefCookieName
	}

	return nil
}
