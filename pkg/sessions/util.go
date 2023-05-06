package sessions

import (
	"encoding/gob"

	"github.com/yubo/apiserver/pkg/authentication/user"
)

func init() {
	gob.Register(&user.DefaultInfo{})
}

func UserFrom(sess Session) *user.DefaultInfo {
	user, _ := sess.Get(UserInfoKey).(*user.DefaultInfo)
	return user
}

func WithUser(sess Session, u *user.DefaultInfo) error {
	sess.Set(sess, UserInfoKey)
	return sess.Save()
}
