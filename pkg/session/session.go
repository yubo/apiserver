package session

import (
	"net/http"
	"strings"

	"github.com/yubo/apiserver/pkg/authentication/authenticator"
	"github.com/yubo/apiserver/pkg/authentication/user"
	"github.com/yubo/apiserver/pkg/request"
)

type Authenticator struct{}

func NewAuthenticator() authenticator.Request {
	return &Authenticator{}
}

func (a *Authenticator) AuthenticateRequest(req *http.Request) (*authenticator.Response, bool, error) {
	if sess, ok := request.SessionFrom(req.Context()); ok {
		return &authenticator.Response{
			User: &user.DefaultInfo{
				Name:   sess.Get("username"),
				Groups: append(strings.Split(sess.Get("groups"), ","), user.AllAuthenticated),
			},
		}, true, nil
	}

	return nil, false, nil

}
func (a *Authenticator) Name() string {
	return "session authenticator"
}

func (a *Authenticator) Priority() int {
	return authenticator.PRI_TOKEN_OIDC
}

func (a *Authenticator) Available() bool {
	return true
}
