package main

import (
	"encoding/base64"
	"fmt"
	"io"
	"os"

	"github.com/yubo/apiserver/pkg/proc"
	"golang.org/x/net/websocket"
	"k8s.io/klog/v2"
)

const (
	fakeToken            = "1234"
	bearerProtocolPrefix = "base64url.bearer.authorization.k8s.io."
)

func main() {
	os.Exit(proc.PrintErrln(run()))
}

// Sec-WebSocket-Protocol: base64.RawURLEncoding.DecodeString(encodedToken)
// websocket
func run() error {
	token := fakeToken
	if len(os.Args) > 1 {
		token = os.Args[1]
	}

	protocol := bearerProtocolPrefix + base64.RawURLEncoding.EncodeToString([]byte(token))
	config, err := websocket.NewConfig("ws://127.0.0.1:8080/hello", "http://127.0.0.1:8080/hello")
	if err != nil {
		return err
	}
	config.Protocol = []string{
		protocol,
		"dummy",
	}
	ws, err := websocket.DialConfig(config)
	if err != nil {
		return fmt.Errorf("websocket Dial err %s", err)
	}

	for {
		b, err := wsRead(ws)
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		klog.InfoS("recv", "msg", b)
	}
}

func wsRead(conn *websocket.Conn) ([]byte, error) {
	for {
		var data []byte
		err := websocket.Message.Receive(conn, &data)
		if err != nil {
			return nil, err
		}

		if len(data) == 0 {
			continue
		}

		return data, err
	}
}
