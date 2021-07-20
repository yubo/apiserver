package rest

const (
	// Accept or Content-Type used in Consumes() and/or Produces()
	MIME_JSON        = "application/json"
	MIME_XML         = "application/xml"
	MIME_TXT         = "text/plain"
	MIME_URL_ENCODED = "application/x-www-form-urlencoded"
	MIME_OCTET       = "application/octet-stream" // If Content-Type is not present in request, use the default

	PathType   = "path"
	QueryType  = "query"
	HeaderType = "header"

	MaxFormSize = int64(1<<63 - 1)
)
