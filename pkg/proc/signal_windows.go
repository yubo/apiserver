package proc

import (
	"os"
)

var shutdownSignal = os.Interrupt
var shutdownSignals = []os.Signal{os.Interrupt}
var reloadSignals = []os.Signal{}
