package protobuf

import (
	"fmt"
	"io/ioutil"
	"reflect"

	"github.com/emicklei/go-restful"
	"github.com/gogo/protobuf/proto"
)

const MIME_PROTOBUF = "application/x-protobuf" // Accept or Content-Type used in Consumes() and/or Produces()

// NewEntityAccessorMPack returns a new EntityReaderWriter for accessing MessagePack content.
// This package is not initialized with such an accessor using the MIME_MSGPACK contentType.
func NewEntityAccessorProtobuf() restful.EntityReaderWriter {
	return entityProtobufAccess{}
}

// entityOctetAccess is a EntityReaderWriter for Octet encoding
type entityProtobufAccess struct {
}

// Read unmarshalls the value from byte slice and using msgpack to unmarshal
func (e entityProtobufAccess) Read(req *restful.Request, v interface{}) error {
	pb, ok := v.(proto.Message)
	if !ok {
		return fmt.Errorf("%s is not a protobuf message", reflect.TypeOf(v).String())
	}

	b, err := ioutil.ReadAll(req.Request.Body)
	if err != nil {
		return err
	}
	return proto.Unmarshal(b, pb)
}

// Write marshals the value to byte slice and set the Content-Type Header.
func (e entityProtobufAccess) Write(resp *restful.Response, status int, v interface{}) error {
	if v == nil {
		resp.WriteHeader(status)
		// do not write a nil representation
		return nil
	}
	resp.WriteHeader(status)

	pb, ok := v.(proto.Message)
	if !ok {
		return fmt.Errorf("%s is not a protobuf message", reflect.TypeOf(v).String())
	}

	b, err := proto.Marshal(pb)
	if err != nil {
		return err
	}

	_, err = resp.Write(b)
	return err
}
