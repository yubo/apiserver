package db

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/yubo/golib/api"
	"github.com/yubo/golib/configer"
	"github.com/yubo/golib/util"
)

type DBConfig struct {
	Name            string       `json:"name"`
	Driver          string       `json:"driver"`
	Dsn             string       `json:"dsn"`
	MaxRows         int          `json:"maxRows"` // default orm.DefaultRows=1000
	WithoutPing     bool         `json:"withoutPing"`
	IgnoreNotFound  bool         `json:"ignoreNotFound"`
	MaxIdleCount    int          `json:"maxIdleCount"`
	MaxOpenConns    int          `json:"maxOpenConns"`
	ConnMaxLifetime api.Duration `json:"connMaxLifetime"`
	ConnMaxIdletime api.Duration `json:"connMaxIdletime"`
	AutoMigrate     bool         `json:"autoMigrate" description:"auto migrate"`
}

func (p *DBConfig) Validate() error {
	if p.Dsn == "" {
		return nil
	}
	if p.Name == "" {
		return fmt.Errorf("name must be set")
	}
	if p.Driver == "" {
		p.Driver = "mysql"
	}
	return nil
}

type Config struct {
	Debug bool `json:"debug"`
	DBConfig
	Databases []DBConfig `json:"databases"`
}

func (p *Config) GetTags() map[string]*configer.FieldTag {
	return map[string]*configer.FieldTag{
		"driver": {Flag: []string{"db-driver"}, Default: "mysql", Description: "database drivers. Comma-delimited list of:" + strings.Join(sql.Drivers(), ",") + "."},
		"dsn":    {Flag: []string{"db-dsn"}, Description: "disabled if empty. e.g. \n  mysql: 'root:1234@tcp(127.0.0.1:3306)/test?parseTime=truer'\n  sqlite3: 'file:test.db?cache=shared&mode=memory'"},
	}
}

func (p Config) String() string {
	return util.Prettify(p)
}

func (p *Config) Validate() error {
	p.Name = DefaultName
	p.Databases = append(p.Databases, p.DBConfig)

	for i := range p.Databases {
		if err := p.Databases[i].Validate(); err != nil {
			return fmt.Errorf("db[%d]: %s", i, err)
		}
	}
	return nil
}
