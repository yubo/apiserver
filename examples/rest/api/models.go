package api

import (
	"time"
)

type User struct {
	Name      *string    `json:"name" sql:",where,primary_key,size=32" description:"user name"`
	Age       *int       `json:"age" description:"user age"`
	CreatedAt *time.Time `json:"createdAt" description:"created at"`
	UpdatedAt *time.Time `json:"updatedAt" description:"updated at"`
}
