// +build darwin

package rsh

import (
	"errors"
	"os"
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

	if err = _grantpt(pty); err != nil {
		return
	}

	if err = _unlockpt(pty); err != nil {
		return
	}

	tty, err = os.OpenFile(name, os.O_RDWR, 0)
	return
}

func _ptsname(f *os.File) (string, error) {
	//n := make([]byte, _IOC_PARM_LEN(syscall.TIOCPTYGNAME))
	n := make([]byte, syscall.TIOCPTYGNAME)

	err := ioctl(f.Fd(), syscall.TIOCPTYGNAME, uintptr(unsafe.Pointer(&n[0])))
	if err != nil {
		return "", err
	}

	for i, c := range n {
		if c == 0 {
			return string(n[:i]), nil
		}
	}
	return "", errors.New("TIOCPTYGNAME string not NUL-terminated")
}

func _grantpt(f *os.File) error {
	return ioctl(f.Fd(), syscall.TIOCPTYGRANT, 0)
}

func _unlockpt(f *os.File) error {
	return ioctl(f.Fd(), syscall.TIOCPTYUNLK, 0)
}
