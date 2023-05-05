package cookie

import (
	"context"

	gsessions "github.com/gorilla/sessions"
	"github.com/yubo/apiserver/pkg/sessions"
	sessionsr "github.com/yubo/apiserver/pkg/sessions/register"
)

const (
	storeType = "cookie"
)

type Store interface {
	sessions.Store
}

// Keys are defined in pairs to allow key rotation, but the common case is to set a single
// authentication key and optionally an encryption key.
//
// The first key in a pair is used for authentication and the second for encryption. The
// encryption key can be set to nil or omitted in the last pair, but the authentication key
// is required in all pairs.
//
// It is recommended to use an authentication key with 32 or 64 bytes. The encryption key,
// if set, must be either 16, 24, or 32 bytes to select AES-128, AES-192, or AES-256 modes.
func NewStore(options *sessions.Options) Store {
	s := &store{
		options:     options,
		CookieStore: gsessions.NewCookieStore(options.KeyPairs...),
	}
	s.CookieStore.Options = options.ToGorillaOptions()
	return s
}

type store struct {
	options *sessions.Options

	*gsessions.CookieStore
}

func (c *store) Name() string {
	return c.options.Name
}

func (c *store) Type() string {
	return storeType
}

func factory(ctx context.Context, option *sessions.Options) (sessions.Store, error) {
	store := NewStore(option)
	return store, nil
}

func init() {
	sessionsr.RegisterStore(storeType, factory)
}
