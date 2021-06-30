package rsh

import (
	"context"
	"io"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/yubo/apiserver/pkg/rest"
	"github.com/yubo/golib/staging/util/interrupt"
	"github.com/yubo/golib/staging/util/term"
	"k8s.io/klog/v2"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second
	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second
	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10
	// Maximum message size allowed from peer.
	maxMessageSize = 1024
)

// IOStreams provides the standard names for iostreams.  This is useful for embedding and for unit testing.
// Inconsistent and different names make it hard to read and review code
type IOStreams struct {
	In     io.Reader // In think, os.Stdin
	Out    io.Writer // Out think, os.Stdout
	ErrOut io.Writer // ErrOut think, os.Stderr
}

func (p *IOStreams) Read(b []byte) (int, error) {
	return p.In.Read(b)
}

func (p *IOStreams) Write(b []byte) (int, error) {
	return p.Out.Write(b)
}

type Conn struct {
	*websocket.Conn
	tx sync.Mutex
	rx sync.Mutex
}

func (c *Conn) WriteMessage(messageType int, data []byte) error {
	c.tx.Lock()
	defer c.tx.Unlock()
	return c.Conn.WriteMessage(messageType, data)
}

func (c *Conn) ReadMessage() (messageType int, p []byte, err error) {
	c.rx.Lock()
	defer c.rx.Unlock()
	return c.Conn.ReadMessage()
}

type Client struct {
	IOStreams
	Opt             *rest.RequestOptions
	Stdin           bool
	TTY             bool
	Quiet           bool               // minimize unnecessary output
	InterruptParent *interrupt.Handler // InterruptParent, if set, is used to handle interrupts while attached
	lastMsg         string             // TODO: remove it
	err             error

	overrideStreams func() (io.ReadCloser, io.Writer, io.Writer)
	isTerminalIn    func(t term.TTY) bool
	SizeQueue       <-chan term.TerminalSize
	ctx             context.Context
	cancel          context.CancelFunc
	conn            Conn
}

func (p *Client) Setup() term.TTY {
	t := term.TTY{
		Parent: p.InterruptParent,
		Out:    p.Out,
	}

	if !p.Stdin {
		// need to nil out p.In to make sure we don't create a stream for stdin
		p.In = nil
		p.TTY = false
		return t
	}

	t.In = p.In
	if !p.TTY {
		return t
	}

	if p.isTerminalIn == nil {
		p.isTerminalIn = func(tty term.TTY) bool {
			return tty.IsTerminalIn()
		}
	}
	if !p.isTerminalIn(t) {
		p.TTY = false
		klog.V(5).Info("Unable to use a TTY - input is not a terminal or the right kind of file")
		return t
	}

	// if we get to here, the user wants to attach stdin, wants a TTY, and p.In is a terminal, so we
	// can safely set t.Raw to true
	t.Raw = true

	if p.overrideStreams == nil {
		// use dockerterm.StdStreams() to get the right I/O handles on Windows
		p.overrideStreams = StdStreams
	}
	stdin, stdout, _ := p.overrideStreams()
	p.In = stdin
	t.In = stdin
	if p.Out != nil {
		p.Out = stdout
		t.Out = stdout
	}

	return t
}

// StdStreams returns the standard streams (stdin, stdout, stderr).
func StdStreams() (stdIn io.ReadCloser, stdOut, stdErr io.Writer) {
	return os.Stdin, os.Stdout, os.Stderr
}
