// this is a sample echo rest api module
package user

import (
	"github.com/yubo/apiserver/pkg/rest"
)

type User struct {
	Id    int    `json:"id"`
	Name  string `json:"name"`
	Phone string `json:"phone"`
}

type CreateUserInput struct {
	Name  string  `json:"name"`
	Phone *string `json:"phone"`
}

type CreateUserOutput struct {
	ID int64 `json:"id" description:"id"`
}

type ListUserInput struct {
	rest.Pagination
	Query *string `param:"query" name:"query" description:"query user"`
	Count bool    `param:"query" name:"count" description:"just response total count"`
}

type ListUserOutput struct {
	Total int     `json:"total"`
	List  []*User `json:"list"`
}

type GetUserInput struct {
	Name string `param:"path" name:"user-name"`
}

func (p *GetUserInput) Validate() error {
	return nil
}

type UpdateUserParam struct {
	Name string `param:"path" name:"user-name"`
}

type UpdateUserBody struct {
	Name  string `json:"-" sql:",where"`
	Phone string `json:"phone"`
}

type DeleteUserInput struct {
	Name string `param:"path" name:"user-name"`
}
