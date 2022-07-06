package register

import (
	s3 "github.com/yubo/apiserver/pkg/s3/module"
)

func init() {
	s3.Register()
}
