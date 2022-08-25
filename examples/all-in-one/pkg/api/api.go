// this is a sample echo rest api module
package api

import (
	"time"

	"github.com/yubo/apiserver/pkg/rest"
)

type User struct {
	Name      string `sql:",where,primary_key,size=32"`
	Age       int
	CreatedAt time.Time
	UpdatedAt time.Time
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
	rest.PageParams
	Query *string `param:"query" name:"query" description:"query user"`
}

type ListUserOutput struct {
	Total int64  `json:"total"`
	List  []User `json:"list"`
}

type GetUserParam struct {
	Name string `param:"path" name:"name"`
}

func (p *GetUserParam) Validate() error {
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

type DeleteUserParam struct {
	Name string `param:"path" name:"name"`
}
