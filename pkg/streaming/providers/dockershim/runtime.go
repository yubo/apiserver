package dockershim

import (
	"time"

	"github.com/yubo/apiserver/pkg/streaming"
	"github.com/yubo/apiserver/pkg/streaming/providers/dockershim/libdocker"
)

func NewRuntime(dockerEndpoint string, requestTimeout, imagePullTimeout time.Duration) streaming.Runtime {
	return &streamingRuntime{
		client: libdocker.ConnectToDockerOrDie(
			dockerEndpoint,
			requestTimeout,
			imagePullTimeout,
		),
		execHandler: &NativeExecHandler{},
	}
}

func NewProvider(dockerEndpoint string, requestTimeout, imagePullTimeout time.Duration) streaming.Provider {
	return streaming.NewProvider(NewRuntime(dockerEndpoint, requestTimeout, imagePullTimeout))
}
