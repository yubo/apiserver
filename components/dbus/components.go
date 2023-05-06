package dbus

import (
	"context"

	authUser "github.com/yubo/apiserver/pkg/authentication/user"
)

type key int

const (
	passwordfileKey key = iota
)

type Passwordfile interface {
	Authenticate(ctx context.Context, usr, pwd string) authUser.Info
}

func RegisterPasswordfile(v Passwordfile) {
	MustRegister(passwordfileKey, v)
}

func GetPasswordfile() (Passwordfile, bool) {
	v, ok := Get(passwordfileKey)
	if !ok {
		return nil, false
	}

	v2, ok := v.(Passwordfile)
	return v2, ok
}
