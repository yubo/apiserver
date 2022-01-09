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
		SidLength:      32,
		HttpOnly:       true,
		Storage:        "db",
		GcInterval:     api.NewDuration("600s"),
		CookieLifetime: api.NewDuration("16h"),
		MaxIdleTime:    api.NewDuration("1h"),
		TableName:      "session",
		CookieName:     DefCookieName,
	}
}

type Config struct {
	CookieName     string
	SidLength      int
	HttpOnly       bool
	AutoMigrate    bool
	Domain         string
	GcInterval     api.Duration
	CookieLifetime api.Duration
	MaxIdleTime    api.Duration
	DBName         string `json:"dbName"`
	Storage        string `description:"mem|db(defualt)"`
	TableName      string `json:"tableName"`
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

	if p.Storage != "db" && p.Storage != "mem" {
		return fmt.Errorf("storage %s is invalid, should be mem or db(default)", p.Storage)
	}

	return nil
}
