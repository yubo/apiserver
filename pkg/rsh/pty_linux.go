// +build linux

package rsh

import (
	"os"
	"strconv"
	"syscall"
	"unsafe"
)

func newPty() (pty, tty *os.File, name string, err error) {
	if pty, err = os.OpenFile("/dev/ptmx", os.O_RDWR, 0); err != nil {
		return
	}

	defer func() {
		if err != nil {
			pty.Close() // Best effort.
		}
	}()

	if name, err = _ptsname(pty); err != nil {
		return
	}

	if err = _unlockpt(pty); err != nil {
		return
	}

	tty, err = os.OpenFile(name, os.O_RDWR|syscall.O_NOCTTY, 0)
	return
}

func _ptsname(pty *os.File) (string, error) {
	var n int32
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, pty.Fd(),
		syscall.TIOCGPTN, uintptr(unsafe.Pointer(&n))); errno != 0 {
		return "", errno
	}
	return "/dev/pts/" + strconv.Itoa(int(n)), nil
}

func _unlockpt(pty *os.File) error {
	var n int32
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, pty.Fd(),
		syscall.TIOCSPTLCK, uintptr(unsafe.Pointer(&n)))
	if errno != 0 {
		pty.Close()
		return errno
	}
	return nil
}
