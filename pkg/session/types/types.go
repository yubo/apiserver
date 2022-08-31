package types

import (
	"net/http"
	"net/url"
	"time"
)

type SessionConn struct {
	Sid        string `sql:",where,primary_key"`
	UserName   string
	Data       url.Values
	CookieName string
	CreatedAt  int64
	UpdatedAt  int64 `sql:",index"`
}

type Session interface {
	Get(key string) string
	GetValues() map[string][]string
	Set(key, value string) error
	Add(key, value string) error
	Del(key string)
	Has(key string) bool

	Reset() error
	Sid() string
	Update(w http.ResponseWriter) error
	CreatedAt() time.Time
}
