package options

const (
	_ uint16 = iota << 3

	PRI_M_DB      // no  dep
	PRI_M_AUTHN   // no  dep
	PRI_M_SESSION // dep http
	PRI_M_AUTHZ   // dep authn
	PRI_M_SWAGGER // dep http
	PRI_M_HTTP    // dep authn authz
	PRI_M_GRPC    // dep tracing authn authz
	PRI_M_TRACING // dep http
)
