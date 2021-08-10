// this is a sample echo rest api module
package user

import (
	"github.com/yubo/apiserver/pkg/rest"
	"github.com/yubo/golib/util"
)

type User struct {
	Id    int    `json:"id"`
	Name  string `json:"name"`
	Phone string `json:"phone"`
}

type CreateUserInput struct {
	Name  *string `json:"name"`
	Phone *string `json:"phone"`
}

type CreateUserOutput rest.RespID

type GetUsersInput struct {
	rest.Pagination
	Query *string `param:"query" name:"query" description:"query user"`
	Count bool    `param:"query" name:"count" description:"just response total count"`
}

func (p *GetUsersInput) Validate() error {
	return nil
}

func (p GetUsersInput) String() string {
	return util.Prettify(p)
}

type GetUsersOutput struct {
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

type DeleteUserOutput struct {
	rest.RespStatus
	Data *User `json:"dat"`
}