package session

import (
	"net/http"

	"github.com/yubo/apiserver/pkg/authentication/authenticator"
	"github.com/yubo/apiserver/pkg/authentication/user"
	"github.com/yubo/apiserver/pkg/sessions"
)

type Authenticator struct{}

func NewAuthenticator() authenticator.Request {
	return &Authenticator{}
}

func (a *Authenticator) AuthenticateRequest(req *http.Request) (*authenticator.Response, bool, error) {
	sess, ok := sessions.SessionFrom(req.Context())
	if !ok {
		return nil, false, nil
	}

	user := &user.DefaultInfo{}
	if user.Name, _ = sess.Get("userName").(string); user.Name == "" {
		return nil, false, nil
	}
	user.Groups, _ = sess.Get("groups").([]string)
	user.Extra, _ = sess.Get("extra").(map[string][]string)

	return &authenticator.Response{User: user}, true, nil
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
