//go:build !windows
// +build !windows

package proc

import (
	"os"
	"syscall"
)

var shutdownSignal = os.Interrupt
var shutdownSignals = []os.Signal{os.Interrupt, syscall.SIGTERM}
var reloadSignals = []os.Signal{}
