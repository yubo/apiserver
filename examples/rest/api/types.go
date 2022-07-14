package api

import (
	"time"

	"github.com/yubo/apiserver/pkg/rest"
)

type User struct {
	Name      string    `sql:",where,primary_key,size=32" json:"name" description:"user name"`
	Age       int       `json:"age" description:"user age"`
	CreatedAt time.Time `json:"createdAt" description:"created at"`
	UpdatedAt time.Time `json:"updatedAt" description:"updated at"`
}

type CreateUserInput struct {
	Name string `sql:",where"`
	Age  int
}

func (p *CreateUserInput) User() *User {
	return &User{
		Name: p.Name,
		Age:  p.Age,
	}
}

type ListInput struct {
	rest.Pagination
	Query *string `param:"query" name:"query" description:"query user"`
}

type ListUserOutput struct {
	List        []User `json:"list"`
	CurrentPage int    `json:"currentPage"`
	PageSize    int    `json:"pageSize"`
	Total       int64  `json:"total"`
}

type GetUserInput struct {
	Name string `param:"path" name:"name"`
}

func (p *GetUserInput) Validate() error {
	return nil
}

type UpdateUserParam struct {
	Name string `param:"path" name:"name"`
}

type UpdateUserInput struct {
	Name      string    `json:"-" sql:",where"` // from UpdateUserParam
	Age       *int      `json:"age"`
	UpdatedAt time.Time `json:"-"`
}

type DeleteUserInput struct {
	Name string `param:"path" name:"name"`
}
