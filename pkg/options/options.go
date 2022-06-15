package options

const (
	_ uint16 = iota << 3

	PRI_M_DB      // no  dep
	PRI_M_AUDIT   // no  dep
	PRI_M_HTTP    // no dep
	PRI_M_AUTHN   // dep authn_mode HTTP1
	PRI_M_AUTHZ   // dep authn
	PRI_M_HTTP2   // dep authn authz audit
	PRI_M_TRACING // dep HTTP2
	PRI_M_GRPC    // dep tracing authn authz audit
)
