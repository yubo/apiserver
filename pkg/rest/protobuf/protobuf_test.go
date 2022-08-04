package protobuf

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yubo/apiserver/pkg/rest/protobuf/testdata"
	"github.com/yubo/golib/util"
)

func TestSerializer(t *testing.T) {
	var want, got testdata.User

	writer := &bytes.Buffer{}
	target := NewSerializer()

	want.Name = util.String("name")
	want.Age = util.Int32(16)

	err := target.Encode(&want, writer)
	assert.NoError(t, err)

	_, err = target.Decode(writer.Bytes(), &got)
	assert.NoError(t, err)
	assert.EqualValues(t, &want, &got)
}
