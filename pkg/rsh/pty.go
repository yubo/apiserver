// +build linux darwin

package rsh

import (
	"errors"
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

var ErrUnsupported = errors.New("ErrUnsupported")
var ErrNotImplemented = errors.New("not implemented")

type Pty struct {
	name string
	Pty  *os.File
	Tty  *os.File
	size *WindowSize
}

// don't edit it, used by syscall
type WindowSize struct {
	Height   uint16 // term row
	Width    uint16 // term col
	WidthPx  uint16 // term x pixel
	HeightPx uint16 // term y pixel
}

func NewPty() (p *Pty, err error) {
	p = &Pty{size: &WindowSize{}}
	p.Pty, p.Tty, p.name, err = newPty()
	return
}

func (p *Pty) Name() string {
	return p.name
}

func (p Pty) String() string {
	return fmt.Sprintf("%s Pty(%v) tty(%v) size(%v)",
		p.name, *p.Pty, *p.Tty, *p.size)
}

// Close the devices
func (p *Pty) Close() {
	p.Tty.Close()
	p.Pty.Close()
}

// Resize the pty
func (p *Pty) SizeReset() error {
	return ResizeTerminal(p.Pty.Fd(), int(p.size.Width), int(p.size.Height))
}

func (p *Pty) Resize(width, height uint16) error {
	p.size.Width = width
	p.size.Height = height

	return p.SizeReset()
}

func IsTerminal(b []byte) bool {
	if len(b) != 2 {
		return false
	}

	if b[0] != MsgInput {
		return false
	}

	if b[1] == 3 || b[1] == 4 {
		return true
	}

	return false
}

func ResizeTerminal(fd uintptr, width, height int) error {
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd,
		syscall.TIOCSWINSZ,
		uintptr(unsafe.Pointer(&WindowSize{
			Height: uint16(height),
			Width:  uint16(width)},
		)),
	); errno != 0 {
		return errno
	}

	return nil
}
