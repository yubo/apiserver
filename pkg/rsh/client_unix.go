// +build linux darwin

package rsh

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/yubo/apiserver/pkg/rest"
	"github.com/yubo/golib/staging/util/term"
	"github.com/yubo/golib/util"
	"k8s.io/klog/v2"
)

func (p *Client) Execute(url string, header http.Header, content []byte) (err error) {
	if len(content) > 0 {
		header.Set("Rsh-Data-Length", fmt.Sprintf("%d", len(content)))
		klog.V(5).Infof("Rsh-Data %d", len(content))
	}

	if p.conn.Conn, _, err = websocket.DefaultDialer.Dial(url, header); err != nil {
		klog.Info(err, url, header)
		return
	}

	p.ctx, p.cancel = context.WithCancel(context.Background())
	defer p.conn.Close()

	// connect will closed by remote
	wg := sync.WaitGroup{}
	p.readPump(&wg)
	p.writePump(&wg, content)

	/* Wait for all of the above goroutines to finish */
	wg.Wait()

	klog.V(5).Infof("Client.Loop() exiting")

	return p.Error()
}

func (p *Client) Error() error {
	if p.err != nil {
		return p.err
	}

	// TODO: remove it
	if strings.Contains(p.lastMsg, "exit status 1") ||
		strings.Contains(p.lastMsg, "Error from server") {
		return errors.New(p.lastMsg)
	}

	return nil
}

func RshRequest(opt *rest.RequestOptions) error {
	opt.Method = "GET"
	opt.Url = strings.NewReplacer("http://", "ws://",
		"https://", "wss://").Replace(opt.Url)

	req, err := rest.NewRequest(opt)
	if err != nil {
		klog.Infof("[debug] payload error %s", err.Error())
		return err
	}

	cli := &Client{
		IOStreams: IOStreams{
			In:     os.Stdin,
			Out:    os.Stdout,
			ErrOut: os.Stderr,
		},
		Stdin:     true,
		TTY:       true,
		SizeQueue: make(chan term.TerminalSize, 0),
	}

	t := cli.Setup()
	if m := t.MonitorSize(t.GetSize()); m != nil {
		cli.SizeQueue = m.Ch()
	}

	// klog.V(5).Infof("[debug] %s", req.Curl())

	// unset p.Err if it was previously set because both stdout and stderr go over p.Out when tty is
	// true
	cli.ErrOut = nil

	return t.Safe(func() error {
		return cli.Execute(req.Request.URL.String(), req.Request.Header, req.Content())
	})
}

func (p *Client) readPump(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer func() {
			wg.Done()
			p.cancel()
		}()

		for {
			mt, msg, err := p.conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					klog.V(8).Infof("error: %v", err)
					p.err = err
				}
				return
			}
			switch mt {
			//case websocket.BinaryMessage:
			case websocket.PingMessage:
				if err := p.conn.WriteMessage(websocket.PongMessage, nil); err != nil {
					return
				}
			//case websocket.PongMessage:
			case websocket.CloseMessage:
				return
			case websocket.TextMessage:
				if err := p.msgRx(msg); err != nil {
					klog.Info(err)
					return
				}
			}
		}
	}()
}

func (p *Client) msgRx(msg []byte) error {
	switch msg[0] {
	case MsgOutput:
		p.lastMsg = string(msg[1:])
		p.Out.Write(msg[1:])
	default:
		klog.Infof("Unhandled protocol message: %s", string(msg))
	}
	return nil
}

func (p *Client) writePump(wg *sync.WaitGroup, heloMsg []byte) {
	wg.Add(1)

	if len(heloMsg) > 0 {
		err := p.conn.WriteMessage(websocket.TextMessage, heloMsg)
		klog.V(5).Infof("send hellMsg %d", len(heloMsg))
		if err != nil {
			klog.Errorf("writemessage error %s", err.Error())
		}
	}

	time.Sleep(time.Second)

	go func() {
		defer p.cancel()
		buff := make([]byte, RshBuffSize+1)
		buff[0] = MsgInput

		for {
			n, err := p.In.Read(buff[1:])

			if err == io.EOF {
				buff[1] = 4
				n = 1
				err = nil
			}

			if err != nil {
				return
			}

			if n > 0 {
				if err := p.conn.WriteMessage(websocket.TextMessage, buff[:n+1]); err != nil {
					return
				}
			}
		}
	}()

	go func() {
		ticker := time.NewTicker(pingPeriod)
		defer func() {
			ticker.Stop()
			p.cancel()
			wg.Done()
		}()
		for {
			select {
			case <-ticker.C:
				if err := p.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					return
				}
			case ws := <-p.SizeQueue:
				p.conn.WriteMessage(websocket.TextMessage,
					append([]byte{MsgResize}, util.JsonStr(ws)...))
			case <-p.ctx.Done():
				return
			}
		}
	}()

}
