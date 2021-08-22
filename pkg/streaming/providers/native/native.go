package native

import (
	"io"
	"os"
	"os/exec"

	"github.com/containerd/console"
	"github.com/yubo/apiserver/pkg/streaming"
)

type execConfig struct {
	User         string   // User that will run the command
	Privileged   bool     // Is the container in privileged mode
	Tty          bool     // Attach standard streams to a tty.
	AttachStdin  bool     // Attach the standard input, makes possible user interaction
	AttachStderr bool     // Attach the standard error
	AttachStdout bool     // Attach the standard output
	Detach       bool     // Execute in detach mode
	DetachKeys   string   // Escape keys for detach
	Env          []string // Environment variables
	WorkingDir   string   // Working directory
	Cmd          []string // Execution commands and args

}

// ExecStartCheck is a temp struct used by execStart
// Config fields is part of ExecConfig in runconfig package
type ExecStartCheck struct {
	// ExecStart will first check if it's detached
	//Detach bool
	// Check if there's a tty
	Tty bool
}

// StreamOptions are the options used to configure the stream redirection
type StreamOptions struct {
	RawTerminal  bool
	InputStream  io.Reader
	OutputStream io.Writer
	ErrorStream  io.Writer
	ExecStarted  chan struct{}
}

// ContainerExecInspect holds information returned by exec inspect.
//type ContainerExecInspect struct {
//	ExecID      string `json:"ID"`
//	ContainerID string
//	Running     bool
//	ExitCode    int
//	Pid         int
//}

// ContainerAttachOptions holds parameters to attach to a container.
type AttachOptions struct {
	Stream     bool
	Stdin      bool
	Stdout     bool
	Stderr     bool
	DetachKeys string
	Logs       bool
}

type execResponse struct {
	stdout  io.Reader
	stderr  io.Reader
	stdin   io.Writer
	session *Session

	console console.Console
	slave   *os.File
	cmd     *exec.Cmd

	//Conn   net.Conn
	//Reader *bufio.Reader
}

// TODO
func (p *execResponse) Close() error {
	if p.slave != nil {
		p.slave.Close()
	}
	if p.console != nil {
		p.console.Close()
	}
	return nil
}

func NewRuntime() (streaming.Runtime, error) {
	controller := newController()
	if err := controller.start(); err != nil {
		return nil, err
	}

	return &streamingRuntime{
		controller: controller,
	}, nil
}

func NewProvider() streaming.Provider {
	if runtime, err := NewRuntime(); err != nil {
		panic(err)
	} else {
		return streaming.NewProvider(runtime)
	}
}
