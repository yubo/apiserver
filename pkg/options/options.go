package options

const (
	_ uint16 = iota << 3

	PRI_M_DB      // no  dep
	PRI_M_AUDIT   // no  dep
	PRI_M_AUTHN   // dep authn_mode
	PRI_M_AUTHZ   // dep authn
	PRI_M_HTTP    // dep authn authz audit
	PRI_M_GRPC    // dep tracing authn authz audit
	PRI_M_TRACING // dep http
)
