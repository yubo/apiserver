package server

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yubo/apiserver/pkg/server/urlencoded"
	"github.com/yubo/golib/runtime"
	"github.com/yubo/golib/runtime/serializer"
	"github.com/yubo/golib/runtime/serializer/protobuf"
	"github.com/yubo/golib/runtime/serializer/protobuf/testdata"
	"github.com/yubo/golib/util"
)

func TestCodecFactory(t *testing.T) {
	cf := serializer.NewCodecFactory(protobuf.WithSerializer, urlencoded.WithSerializer)

	contentTypes := []string{MIME_JSON, MIME_PROTOBUF, MIME_YAML, MIME_URL_ENCODED}

	for _, contentType := range contentTypes {
		t.Run(contentType, func(t *testing.T) {
			info, ok := runtime.SerializerInfoForMediaType(cf.SupportedMediaTypes(), contentType)
			assert.Equal(t, true, ok)

			writer := &bytes.Buffer{}
			serializer := info.Serializer

			user := testdata.User{
				Name: util.String("name123"),
				Age:  util.Int32(123),
			}

			err := serializer.Encode(&user, writer)
			assert.NoError(t, err)

			var user2 testdata.User
			_, err = serializer.Decode(writer.Bytes(), &user2)
			assert.NoError(t, err)
			assert.EqualValues(t, &user, &user2)
		})
	}
}
