package rest

const (
	// Accept or Content-Type used in Consumes() and/or Produces()
	MIME_ALL         = "*/*"
	MIME_JSON        = "application/json"
	MIME_XML         = "application/xml"
	MIME_TXT         = "text/plain"
	MIME_URL_ENCODED = "application/x-www-form-urlencoded"
	MIME_PROTOBUF    = "application/x-protobuf"
	MIME_OCTET       = "application/octet-stream" // If Content-Type is not present in request, use the default

	MaxFormSize = int64(1<<63 - 1)

	SecurityDefinitionKey = "OAPI_SECURITY_DEFINITION"
	NativeClientID        = "native-client-id"
	NativeClientSecret    = "native-client-secret"
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
