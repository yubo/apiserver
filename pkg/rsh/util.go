// +build linux darwin

package rsh

import (
	"syscall"
	"time"
)

var (
	RealCrash = false
)

func Nanotime() int64 {
	return time.Now().UnixNano()
}

func crash() {
	if r := recover(); r != nil {
		if RealCrash {
			panic(r)
		}
	}
}

func ioctl(fd, cmd, ptr uintptr) error {
	if _, _, e := syscall.Syscall(syscall.SYS_IOCTL, fd, cmd, ptr); e != 0 {
		return e
	}

	return nil
}
