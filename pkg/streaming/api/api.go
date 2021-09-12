package api

// This file contains all constants defined in CRI.

// Required runtime condition type.
const (
	// RuntimeReady means the runtime is up and ready to accept basic containers.
	RuntimeReady = "RuntimeReady"
	// NetworkReady means the runtime network is up and ready to accept containers which require network.
	NetworkReady = "NetworkReady"
)

// LogStreamType is the type of the stream in CRI container log.
type LogStreamType string

const (
	// Stdout is the stream type for stdout.
	Stdout LogStreamType = "stdout"
	// Stderr is the stream type for stderr.
	Stderr LogStreamType = "stderr"
)

// LogTag is the tag of a log line in CRI container log.
// Currently defined log tags:
// * First tag: Partial/Full - P/F.
// The field in the container log format can be extended to include multiple
// tags by using a delimiter, but changes should be rare. If it becomes clear
// that better extensibility is desired, a more extensible format (e.g., json)
// should be adopted as a replacement and/or addition.
type LogTag string

const (
	// LogTagPartial means the line is part of multiple lines.
	LogTagPartial LogTag = "P"
	// LogTagFull means the line is a single full line or the end of multiple lines.
	LogTagFull LogTag = "F"
	// LogTagDelimiter is the delimiter for different log tags.
	LogTagDelimiter = ":"
)

type ExecRequest struct {
	// ID of the container in which to execute the command.
	ContainerId string `json:"container_id" param:"query" description:"container id"`
	// Command to execute.
	Cmd []string `json:"cmd,omitempty" param:"query"`
	// Whether to exec the command in a TTY.
	Tty bool `json:"tty,omitempty" param:"query"`
	// Whether to stream stdin.
	// One of `stdin`, `stdout`, and `stderr` MUST be true.
	Stdin bool `json:"stdin,omitempty" param:"query"`
	// Whether to stream stdout.
	// One of `stdin`, `stdout`, and `stderr` MUST be true.
	Stdout bool `json:"stdout,omitempty" param:"query"`
	// Whether to stream stderr.
	// One of `stdin`, `stdout`, and `stderr` MUST be true.
	// If `tty` is true, `stderr` MUST be false. Multiplexing is not supported
	// in this case. The output of stdout and stderr will be combined to a
	// single stream.
	Stderr bool `json:"stderr,omitempty" param:"query"`
}

type ExecResponse struct {
	// Fully qualified URL of the exec streaming server.
	Url string `json:"url,omitempty"`
}

type AttachRequest struct {
	// ID of the container to which to attach.
	ContainerId string `json:"container_id,omitempty" param:"query"`
	// Whether to stream stdin.
	// One of `stdin`, `stdout`, and `stderr` MUST be true.
	Stdin bool `json:"stdin,omitempty" param:"query"`
	// Whether the process being attached is running in a TTY.
	// This must match the TTY setting in the ContainerConfig.
	Tty bool `json:"tty,omitempty" param:"query"`
	// Whether to stream stdout.
	// One of `stdin`, `stdout`, and `stderr` MUST be true.
	Stdout bool `json:"stdout,omitempty" param:"query"`
	// Whether to stream stderr.
	// One of `stdin`, `stdout`, and `stderr` MUST be true.
	// If `tty` is true, `stderr` MUST be false. Multiplexing is not supported
	// in this case. The output of stdout and stderr will be combined to a
	// single stream.
	Stderr bool `json:"stderr,omitempty" param:"query"`
}

type AttachResponse struct {
	// Fully qualified URL of the attach streaming server.
	Url string `protobuf:"bytes,1,opt,name=url,proto3" json:"url,omitempty"`
}

type PortForwardRequest struct {
	// ID of the container to which to forward the port.
	PodSandboxId string `json:"pod_sandbox_id,omitempty"`
	// Port to forward.
	Port []int32 `json:"port,omitempty"`
}

type PortForwardResponse struct {
	// Fully qualified URL of the port-forward streaming server.
	Url string `json:"url,omitempty"`
}

type ExecSyncRequest struct {
	// ID of the container.
	ContainerId string `protobuf:"bytes,1,opt,name=container_id,json=containerId,proto3" json:"container_id,omitempty"`
	// Command to execute.
	Cmd []string `protobuf:"bytes,2,rep,name=cmd,proto3" json:"cmd,omitempty"`
	// Timeout in seconds to stop the command. Default: 0 (run forever).
	Timeout              int64    `protobuf:"varint,3,opt,name=timeout,proto3" json:"timeout,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

type ExecSyncResponse struct {
	// Captured command stdout output.
	Stdout []byte `protobuf:"bytes,1,opt,name=stdout,proto3" json:"stdout,omitempty"`
	// Captured command stderr output.
	Stderr []byte `protobuf:"bytes,2,opt,name=stderr,proto3" json:"stderr,omitempty"`
	// Exit code the command finished with. Default: 0 (success).
	ExitCode             int32    `protobuf:"varint,3,opt,name=exit_code,json=exitCode,proto3" json:"exit_code,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}
