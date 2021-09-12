package native

import (
	"time"

	"github.com/yubo/golib/stream"
)

type ExecConfig struct {
	User       string   // User that will run the command
	Detach     bool     // Execute in detach mode
	DetachKeys string   // Escape keys for detach
	Env        []string // Environment variables
	WorkingDir string   // Working directory
	Cmd        []string // Execution commands and args
	Timeout    time.Duration
	recorder   stream.Recorder
}

// ContainerAttachOptions holds parameters to attach to a container.
type AttachOptions struct {
	DetachKeys string
}
