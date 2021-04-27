package session

import (
	"net/http"
	"strings"

	"github.com/yubo/apiserver/pkg/authentication/authenticator"
	"github.com/yubo/apiserver/pkg/authentication/user"
	"github.com/yubo/apiserver/pkg/request"
)

func NewAuthenticator() authenticator.Request {
	return authenticator.RequestFunc(func(req *http.Request) (*authenticator.Response, bool, error) {
		if sess, ok := request.SessionFrom(req.Context()); ok {
			return &authenticator.Response{
				User: &user.DefaultInfo{
					Name:   sess.Get("username"),
					Groups: strings.Split(sess.Get("groups"), ","),
				},
			}, true, nil
		}

		return nil, false, nil
	})
}
