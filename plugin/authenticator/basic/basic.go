package basic

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/yubo/apiserver/pkg/authentication"
	"github.com/yubo/apiserver/pkg/authentication/authenticator"
	"github.com/yubo/apiserver/pkg/authentication/user"
)

func RegisterAuthn(p basicProvider) error {
	if p == nil {
		return fmt.Errorf("invalid basic provider")
	}
	return authentication.RegisterAuthn(newFactory(p))
}

type basicProvider interface {
	Authenticate(user, pwd string) user.Info
}

func newFactory(p basicProvider) func(ctx context.Context) (authenticator.Request, error) {
	return func(ctx context.Context) (authenticator.Request, error) {
		return NewAuthenticator(p), nil
	}
}

type Authenticator struct {
	provider basicProvider
}

func NewAuthenticator(p basicProvider) authenticator.Request {
	return &Authenticator{provider: p}
}

func (a *Authenticator) AuthenticateRequest(r *http.Request) (*authenticator.Response, bool, error) {
	if r.Header.Get("Authorization") == "" {
		return nil, false, nil
	}

	s := strings.SplitN(r.Header.Get("Authorization"), " ", 2)
	if len(s) != 2 || s[0] != "Basic" {
		return nil, false, nil
	}

	b, err := base64.StdEncoding.DecodeString(s[1])
	if err != nil {
		return nil, false, err
	}
	pair := strings.SplitN(string(b), ":", 2)
	if len(pair) != 2 {
		return nil, false, errors.New("Invalid authorization message")
	}

	// Decode the client_id and client_secret pairs as per
	// https://tools.ietf.org/html/rfc6749#section-2.3.1
	username, err := url.QueryUnescape(pair[0])
	if err != nil {
		return nil, false, err
	}

	password, err := url.QueryUnescape(pair[1])
	if err != nil {
		return nil, false, err
	}

	usr := a.provider.Authenticate(username, password)
	if usr == nil {
		return nil, false, err
	}

	return &authenticator.Response{User: usr}, true, nil
}
