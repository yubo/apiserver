// +build !linux,!darwin

package rsh

import (
	"io"
	"os"
	"os/exec"

	"github.com/yubo/apiserver/pkg/openapi"
	"github.com/yubo/golib/util"
)

func (p *Rsh) Run(remote io.ReadWriteCloser, cmds []string, env []string) error {
	return util.ErrUnsupported

}

func RshRequest(opt *openapi.RequestOption) error {
	return util.ErrUnsupported
}

func ResizeTerminal(fd uintptr, width, height int) error {
	return util.ErrUnsupported
}

func newPty() (pty, tty *os.File, name string, err error) {
	return nil, nil, "", util.ErrUnsupported
}

type Player struct{}

func NewPlayer(filename string, speed int64, repeat bool, wait int64) (*Player, error) {
	return nil, util.ErrUnsupported
}
func (p *Player) Close() error { return nil }
func (p *Player) Read(d []byte) (n int, err error) {
	return 0, nil
}
func (p *Player) Write(b []byte) (n int, err error) {
	return 0, nil
}

func RecordRun(cmd *exec.Cmd, fileName string, input bool) error { return util.ErrUnsupported }
