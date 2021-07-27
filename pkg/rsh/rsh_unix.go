// +build linux darwin

package rsh

import (
	"encoding/json"
	"errors"
	"io"
	"os/exec"
	"syscall"

	"github.com/yubo/golib/util/term"
	"k8s.io/klog/v2"
)

func (p *Rsh) Run(remote io.ReadWriter, cmds []string, env []string) error {
	if len(cmds) == 0 {
		return errors.New("empty command")
	}

	cmd := exec.Command(cmds[0], cmds[1:]...)

	if len(env) > 0 {
		cmd.Env = append(p.Env, env...)
	}

	pty, err := NewPty()
	if err != nil {
		return err
	}
	defer pty.Close()

	cmd.Stdout = pty.Tty
	cmd.Stdin = pty.Tty
	cmd.Stderr = pty.Tty
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	// cmd.SysProcAttr.Setsid = true
	// cmd.SysProcAttr.Setctty = true
	// cmd.SysProcAttr.Ctty = int(pty.Tty.Fd())

	p.conn = remote
	p.resize = pty.Resize

	fdx, err := NewFdx(remote, pty.Pty, p.BufferSize)
	if err != nil {
		return err
	}

	fdx.RxFilter(p.RxFilter)
	fdx.TxFilter(p.TxFilter)

	go func() {
		if err := fdx.Run(); err != nil {
			klog.Infof("fdx run return, I/O closed")
		}
		// try close remote connect and local command
		// remote.Close() // remote.Close should be close by conn.Close
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
	}()

	return cmd.Run()
}

// conn <- pty
func (p *Rsh) RxFilter(data []byte) ([]byte, error) {
	return append([]byte{MsgOutput}, data...), nil
}

// conn -> pty
func (p *Rsh) TxFilter(data []byte) ([]byte, error) {

	switch data[0] {
	case MsgInput:
		if !p.PermitWrite {
			return []byte{}, nil
		}
		return data[1:], nil
	case MsgResize:
		var wSize term.TerminalSize
		err := json.Unmarshal(data[1:], &wSize)
		if err != nil {
			return []byte{}, nil
		}
		klog.Infof("resize(%d, %d)", wSize.Width, wSize.Height)
		p.resize(wSize.Width, wSize.Height)
		return []byte{}, nil
	case MsgPing:
		return []byte{}, nil
	case MsgAction:
		if p.action != nil {
			if out, err := p.action(data[1:]); err == nil {
				write(p.conn, append([]byte{MsgAction}, out...))
			}
		}
		return []byte{}, nil
	default:
		klog.Infof("unkown option '%c'", data[0])
		return []byte{}, nil
	}
}
