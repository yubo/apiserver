package rest

import "github.com/yubo/apiserver/pkg/rest/parametercodec"

const (
	// Accept or Content-Type used in Consumes() and/or Produces()
	MIME_ALL         = "*/*"
	MIME_JSON        = "application/json"
	MIME_YAML        = "application/yaml"
	MIME_XML         = "application/xml"
	MIME_TXT         = "text/plain"
	MIME_URL_ENCODED = "application/x-www-form-urlencoded"
	MIME_PROTOBUF    = "application/x-protobuf"   // Accept or Content-Type used in Consumes() and/or Produces()
	MIME_OCTET       = "application/octet-stream" // If Content-Type is not present in request, use the default

	MaxFormSize = int64(1<<63 - 1)

	SecurityDefinitionKey = "OAPI_SECURITY_DEFINITION"
	NativeClientID        = "native-client-id"
	NativeClientSecret    = "native-client-secret"
)

type SecurityType string

const (
	SecurityTypeBase        SecurityType = "base"
	SecurityTypeBearer      SecurityType = "bearer"
	SecurityTypeAPIKey      SecurityType = "apiKey"
	SecurityTypeImplicit    SecurityType = "implicit"
	SecurityTypePassword    SecurityType = "password"
	SecurityTypeApplication SecurityType = "application"
	SecurityTypeAccessCode  SecurityType = "accessCode" // same as oauth2
)

var (
	ParameterCodec = parametercodec.New()
)

func init() {
	ResponseWriterRegister(DefaultRespWriter)
}
