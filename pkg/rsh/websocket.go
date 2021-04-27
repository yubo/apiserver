// +build linux darwin

package rsh

import (
	"io"
	"net/http"
	"time"

	restful "github.com/emicklei/go-restful"
	"github.com/gorilla/websocket"
	"k8s.io/klog/v2"
)

const (
	RshErr = "X-RSH-ERROR:"
)

func NewWebSocket(req *restful.Request, resp *restful.Response, timeout int64) (*RshConn, error) {
	upgrader := &websocket.Upgrader{
		ReadBufferSize:  RshBuffSize,
		WriteBufferSize: RshBuffSize,
		Subprotocols:    []string{"rsh"},
		CheckOrigin:     func(r *http.Request) bool { return true },
	}

	conn, err := upgrader.Upgrade(resp, req.Request, resp.Header())
	if err != nil {
		return nil, err
	}

	return &RshConn{Conn: conn, timeout: time.Second * time.Duration(timeout)}, nil
}

func (p *RshConn) Close() error {
	if len(p.closeMessage) > 0 {
		if err := p.WriteControl(websocket.CloseMessage, p.closeMessage,
			time.Time{}); err == websocket.ErrCloseSent {
			klog.V(3).Infof("ws sent close message err %v", err)
		}
	}
	return p.Conn.Close()
}

func (p *RshConn) Write(data []byte) (n int, err error) {
	p.tx.Lock()
	defer p.tx.Unlock()

	writer, err := p.NextWriter(websocket.TextMessage)
	if err != nil {
		return 0, err
	}
	defer writer.Close()
	return writer.Write(data)
}

func (p *RshConn) Read(data []byte) (n int, err error) {
	p.rx.Lock()
	defer p.rx.Unlock()
	if p.timeout > 0 {
		p.SetReadDeadline(time.Now().Add(p.timeout))
	}
	for {
		if p.reader == nil {
			var messageType int
			messageType, p.reader, err = p.NextReader()
			if err != nil {
				klog.V(5).Infof("p.NextReader err %v", err.Error())
				return 0, err
			}
			if messageType != websocket.TextMessage {
				continue
			}
		}

		n, err = p.reader.Read(data)

		if err == nil {
			return n, nil
		}

		// the current message is end
		// then set reader nil to get next message reader
		if err == io.EOF {
			p.reader = nil
			return n, nil
		}

		klog.Errorf("reader unknown error %s, throw it", err.Error())
		return n, err
	}
}

func (p *RshConn) MsgOutput(msg string) (n int, err error) {
	return p.Write(append([]byte{MsgOutput}, []byte(msg)...))
}

func (p *RshConn) Error(code int, err string) {
	p.MsgOutput(err + "\r\n")
	p.closeMessage = websocket.FormatCloseMessage(code, "")
}
