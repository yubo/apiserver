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
	sess, ok := request.SessionFrom(req.Context())
	if !ok {
		return nil, false, nil
	}
	userName := sess.Get("userName")
	if userName == "" {
		return nil, false, nil
	}

	return &authenticator.Response{User: &user.DefaultInfo{
		Name:   userName,
		Groups: strings.Split(sess.Get("groups"), ","),
	}}, true, nil
}

func (a *Authenticator) Name() string {
	return "session authenticator"
}

func (a *Authenticator) Priority() int {
	return authenticator.PRI_AUTH_SESSION
}

func (a *Authenticator) Available() bool {
	return true
}
