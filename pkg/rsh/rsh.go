package rsh

import (
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	restful "github.com/emicklei/go-restful"
	"github.com/gorilla/websocket"
	"github.com/yubo/golib/util"
)

const (
	MsgInput  byte = '0' + iota // User input typically from a keyboard
	MsgOutput                   // Normal output to the terminal
	MsgResize                   // Notify that the browser size has been changed
	MsgPing
	MsgCtl
	MsgAction // custom Action
)

const (
	rshDataKey = "req-rsh-data"
	rshConnKey = "req-rsh-conn"
)

const (
	RshBuffSize = 1 << 12 // 4096
)

var (
	ErrLocalCommandClosed     = errors.New("pty closed")
	ErrRemoteConnectionClosed = errors.New("connection closed")
)

type RshConfig struct {
	PermitWrite bool
	Timeout     int64 // second
	Env         []string
	BufferSize  int
	TmpPath     string
}

func (p *RshConfig) Validate() error {
	if !util.IsDir(p.TmpPath) {
		return fmt.Errorf("tmp is not dir")
	}
	return nil
}

type Rsh struct {
	*RshConfig
	conn   io.ReadWriter
	resize func(uint16, uint16) error
	action func([]byte) ([]byte, error)
	log    func(format string, args ...interface{})
}

// threadsafe conn
type RshConn struct {
	state        int
	reader       io.Reader
	Data         []byte
	closeMessage []byte
	timeout      time.Duration
	tx           sync.Mutex
	rx           sync.Mutex
	*websocket.Conn
}

func defaultAction([]byte) ([]byte, error) { return []byte{}, nil }

func NewRsh(cf *RshConfig, action func([]byte) ([]byte, error)) (*Rsh, error) {

	rsh := &Rsh{
		RshConfig: cf,
		action:    action,
	}

	if rsh.action == nil {
		rsh.action = defaultAction
	}

	return rsh, nil
}

func WithRshData(r *restful.Request, data []byte) *restful.Request {
	r.SetAttribute(rshDataKey, data)
	return r
}

func RshDataFrom(r *restful.Request) ([]byte, bool) {
	data, ok := r.Attribute(rshDataKey).([]byte)
	return data, ok
}

func WithRshConn(r *restful.Request, conn *RshConn) *restful.Request {
	r.SetAttribute(rshConnKey, conn)
	return r
}

func RshConnFrom(r *restful.Request) (*RshConn, bool) {
	conn, ok := r.Attribute(rshConnKey).(*RshConn)
	return conn, ok
}
