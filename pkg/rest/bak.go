package rest

import "github.com/emicklei/go-restful"

const (
	// scope
	OauthScopeNil           = "nil"
	OauthScopeRead          = "read"
	OauthScopeWrite         = "write"
	OauthScopeExec          = "exec"
	OauthScopeWork          = "work"
	OauthScopeRoot          = "root"
	OauthScopeUpload        = "upload"
	OauthScopeOverwrite     = "overwrite"
	OauthScopeEdit          = "edit"
	OauthScopeAdmin         = "admin"
	OauthScopeReadSecret    = "read:secret"
	OauthScopeWriteSecret   = "write:secret"
	OauthScopeWriteRegistry = "write:registry"
	OauthScopeReadSso       = "read:sso"
	OauthScopeWriteSso      = "write:sso"
	reqToken                = "req-openapi-token"
)

type SecurityType string

const (
	SecurityTypeBase        SecurityType = "base"
	SecurityTypeApiKey      SecurityType = "apiKey"
	SecurityTypeImplicit    SecurityType = "implicity"
	SecurityTypePassword    SecurityType = "password"
	SecurityTypeApplication SecurityType = "application"
	SecurityTypeAccessCode  SecurityType = "accessCode" // same as oauth2
)

type Token interface {
	GetTokenName() string
	GetUserName() string
	HasScope(scope string) bool
}

func TokenFrom(r *restful.Request) (Token, bool) {
	token, ok := r.Attribute(reqToken).(Token)
	return token, ok
}

type AnonymousToken struct{}

func (p AnonymousToken) GetTokenName() string       { return "null" }
func (p AnonymousToken) GetUserName() string        { return "anonymous" }
func (p AnonymousToken) HasScope(scope string) bool { return false }
