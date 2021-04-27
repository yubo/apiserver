package urlencoded

import (
	restful "github.com/emicklei/go-restful"
)

//restful.RegisterEntityAccessor(MIME_URL_ENCODED, NewEntityAccessorUrlEncoded())

// NewEntityAccessorMPack returns a new EntityReaderWriter for accessing MessagePack content.
// This package is not initialized with such an accessor using the MIME_URL_ENCODED contentType.
func NewEntityAccessor() restful.EntityReaderWriter {
	return entityAccess{}
}

// entityAccess is a EntityReaderWriter for post form url encoding
type entityAccess struct {
}

// Read unmarshalls the value from byte slice and using urlencoded to unmarshal
func (e entityAccess) Read(req *restful.Request, v interface{}) error {
	return NewDecoder(req.Request.Body).Form(req.Request.Form).Decode(v)
}

// Write marshals the value to byte slice and set the Content-Type Header.
func (e entityAccess) Write(resp *restful.Response, status int, v interface{}) error {
	resp.WriteHeader(status)

	if v == nil {
		// do not write a nil representation
		return nil
	}
	return NewEncoder(resp).Encode(v)
}
