package dockershim

import (
	"github.com/yubo/golib/util/runtime"
	"github.com/yubo/golib/util/term"
)

// HandleResizing spawns a goroutine that processes the resize channel, calling resizeFunc for each
// remotecommand.TerminalSize received from the channel. The resize channel must be closed elsewhere to stop the
// goroutine.
func HandleResizing(resize <-chan term.TerminalSize, resizeFunc func(size term.TerminalSize)) {
	if resize == nil {
		return
	}

	go func() {
		defer runtime.HandleCrash()

		for size := range resize {
			if size.Height < 1 || size.Width < 1 {
				continue
			}
			resizeFunc(size)
		}
	}()
}
