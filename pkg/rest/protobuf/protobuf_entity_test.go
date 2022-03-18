package protobuf

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/emicklei/go-restful/v3"
	"github.com/stretchr/testify/assert"
)

func TestMsgPack(t *testing.T) {

	// register msg pack entity
	restful.RegisterEntityAccessor(MIME_PROTOBUF, NewEntityAccessorProtobuf())

	// Write
	httpWriter := httptest.NewRecorder()
	msg := &User{Name: "tom", Age: 14}
	resp := restful.NewResponse(httpWriter)
	resp.SetRequestAccepts(MIME_PROTOBUF)

	err := resp.WriteEntity(msg)
	if err != nil {
		t.Errorf("err %v", err)
	}

	// Read
	bodyReader := bytes.NewReader(httpWriter.Body.Bytes())
	httpRequest, _ := http.NewRequest("GET", "/test", bodyReader)
	httpRequest.Header.Set("Content-Type", MIME_PROTOBUF)
	request := restful.NewRequest(httpRequest)
	readMsg := new(User)
	err = request.ReadEntity(readMsg)
	if err != nil {
		t.Errorf("err %v", err)
	}
	assert.Equal(t, msg.Name, readMsg.Name)
	assert.Equal(t, msg.Age, readMsg.Age)
}
