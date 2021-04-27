// +build linux darwin

package rsh

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/creack/pty"
)

type LocalCommand struct {
	command string
	argv    []string

	closeSignal  syscall.Signal
	closeTimeout time.Duration

	cmd *exec.Cmd
	pty *os.File
}

func NewLocalCommand(command []string, env []string) (*LocalCommand, error) {
	if len(command) == 0 {
		return nil, errors.New("empty command")
	}

	cmd := exec.Command(command[0], command[1:]...)

	if len(env) > 0 {
		cmd.Env = append(os.Environ(), env...)
	}

	pty, err := pty.Start(cmd)
	if err != nil {
		// todo close cmd?
		return nil, err
	}

	lcmd := &LocalCommand{
		command: command[0],
		argv:    command[1:],

		cmd: cmd,
		pty: pty,
	}

	// When the process is closed by the user,
	// close pty so that Read() on the pty breaks with an EOF.
	go func() {
		defer func() {
			lcmd.pty.Close()
		}()

		lcmd.cmd.Wait()
	}()

	return lcmd, nil
}

func (lcmd *LocalCommand) Read(p []byte) (n int, err error) {
	return lcmd.pty.Read(p)
}

func (lcmd *LocalCommand) Write(p []byte) (n int, err error) {
	return lcmd.pty.Write(p)
}

func (lcmd *LocalCommand) Close() error {
	return lcmd.cmd.Process.Signal(syscall.SIGKILL)
}

func (lcmd *LocalCommand) WindowTitleVariables() map[string]interface{} {
	return map[string]interface{}{
		"command": lcmd.command,
		"argv":    lcmd.argv,
		"pid":     lcmd.cmd.Process.Pid,
	}
}

func (lcmd *LocalCommand) ResizeTerminal(width int, height int) error {
	return ResizeTerminal(lcmd.pty.Fd(), width, height)
}

func (lcmd *LocalCommand) closeTimeoutC() <-chan time.Time {
	if lcmd.closeTimeout >= 0 {
		return time.After(lcmd.closeTimeout)
	}

	return make(chan time.Time)
}
