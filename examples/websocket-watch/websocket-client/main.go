package main

import (
	"os"

	"golang.org/x/net/websocket"
	"k8s.io/klog/v2"
)

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

// websocket
func ws() error {
	ws, err := websocket.Dial("ws://127.0.0.1:8080/hello", "", "http://127.0.0.1/")
	if err != nil {
		return err
	}

	for {
		b, err := wsRead(ws)
		if err != nil {
			return err
		}
		klog.InfoS("recv", "contain", b)
	}
}

func main() {
	if err := ws(); err != nil {
		klog.Error(err)
		os.Exit(1)
	}
}
