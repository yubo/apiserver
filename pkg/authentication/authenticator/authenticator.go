package authenticator

const (
	_ = iota << 8
	PRI_AUTH_SESSION
	PRI_AUTH_TOKEN
	PRI_AUTH_WEBSOCKET_TOKEN
	PRI_AUTH_ANONYMOUS
)

const (
	_ = iota << 8
	PRI_TOKEN_TEST
	PRI_TOKEN_FILE
	PRI_TOKEN_BOOTSTRAP
	PRI_TOKEN_OIDC
	PRI_TOKEN_CUSTOM
)

type Authn interface {
	APIAudiences() Audiences
	Authenticator() Request
}
