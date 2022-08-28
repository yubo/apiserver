// this is a sample echo rest api module
package api

import (
	"time"
)

type User struct {
	Name      *string `sql:",where,primary_key,size=32"`
	Age       *int
	CreatedAt *time.Time
	UpdatedAt *time.Time
}
