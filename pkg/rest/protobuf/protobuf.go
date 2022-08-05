package protobuf

import (
	"fmt"
	"io"
	"net/http"
	"reflect"

	"github.com/gogo/protobuf/proto"
	"github.com/yubo/golib/api"
	"github.com/yubo/golib/runtime"
	"github.com/yubo/golib/runtime/serializer"
	"github.com/yubo/golib/util/framer"
)

const (
	ContentTypeProtobuf = "application/x-protobuf" // Accept or Content-Type used in Consumes() and/or Produces()
)

func WithSerializer(options *serializer.CodecFactoryOptions) {
	protoSerializer := NewSerializer()
	options.Serializers = append(options.Serializers, serializer.SerializerType{
		AcceptContentTypes: []string{ContentTypeProtobuf},
		ContentType:        ContentTypeProtobuf,
		FileExtensions:     []string{"pb"},
		Serializer:         protoSerializer,
		Framer:             LengthDelimitedFramer,
		StreamSerializer:   protoSerializer,
	})
}

type errNotMarshalable struct {
	t reflect.Type
}

func (e errNotMarshalable) Error() string {
	return fmt.Sprintf("object %v does not implement the protobuf marshalling interface and cannot be encoded to a protobuf message", e.t)
}

func (e errNotMarshalable) Status() api.Status {
	return api.Status{
		Status:  api.StatusFailure,
		Code:    http.StatusNotAcceptable,
		Reason:  api.StatusReason("NotAcceptable"),
		Message: e.Error(),
	}
}

// IsNotMarshalable checks the type of error, returns a boolean true if error is not nil and not marshalable false otherwise
func IsNotMarshalable(err error) bool {
	_, ok := err.(errNotMarshalable)
	return err != nil && ok
}

// NewSerializer creates a Protobuf serializer that handles encoding versioned objects into the proper wire form.
func NewSerializer() *Serializer {
	return &Serializer{}
}

// Serializer handles encoding versioned objects into the proper wire form
type Serializer struct{}

var _ runtime.Serializer = &Serializer{}

func (s *Serializer) Decode(data []byte, into runtime.Object) (runtime.Object, error) {
	pb, ok := into.(proto.Message)
	if !ok {
		return nil, errNotMarshalable{reflect.TypeOf(into)}
	}

	if err := proto.Unmarshal(data, pb); err != nil {
		return nil, err
	}
	return into, nil
}

// Encode serializes the provided object to the given writer.
func (s *Serializer) Encode(obj runtime.Object, w io.Writer) error {
	switch t := obj.(type) {
	case proto.Marshaler:
		// this path performs extra allocations
		data, err := t.Marshal()
		if err != nil {
			return err
		}
		_, err = w.Write(data)
		return err

	default:
		return errNotMarshalable{reflect.TypeOf(obj)}
	}
}

// LengthDelimitedFramer is exported variable of type lengthDelimitedFramer
var LengthDelimitedFramer = lengthDelimitedFramer{}

// Provides length delimited frame reader and writer methods
type lengthDelimitedFramer struct{}

// NewFrameWriter implements stream framing for this serializer
func (lengthDelimitedFramer) NewFrameWriter(w io.Writer) io.Writer {
	return framer.NewLengthDelimitedFrameWriter(w)
}

// NewFrameReader implements stream framing for this serializer
func (lengthDelimitedFramer) NewFrameReader(r io.ReadCloser) io.ReadCloser {
	return framer.NewLengthDelimitedFrameReader(r)
}
