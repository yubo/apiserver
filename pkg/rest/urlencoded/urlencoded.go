package urlencoded

const (
	maxFormSize      = int64(1<<63 - 1)
	MIME_URL_ENCODED = "application/x-www-form-urlencoded" // Accept or Content-Type used in Consumes() and/or Produces()
)
