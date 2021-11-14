package options

import (
	"io"
)

type Client interface {
	GetId() string
	GetSecret() string
	GetRedirectUri() string
}

type Executer interface {
	Execute(wr io.Writer, data interface{}) error
}
