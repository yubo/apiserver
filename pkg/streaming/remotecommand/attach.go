/*
Copyright 2016 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package remotecommand

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/yubo/golib/api"
	apierrors "github.com/yubo/golib/api/errors"
	"github.com/yubo/golib/types"
	"github.com/yubo/golib/util/runtime"
	"github.com/yubo/golib/term"
)

// Attacher knows how to attach to a running container in a pod.
type Attacher interface {
	// AttachContainer attaches to the running container in the pod, copying data between in/out/err
	// and the container's stdin/stdout/stderr.
	AttachContainer(name string, uid types.UID, container string, in io.Reader, out, err io.WriteCloser, tty bool, resize <-chan term.TerminalSize) error
}

// ServeAttach handles requests to attach to a container. After creating/receiving the required
// streams, it delegates the actual attaching to attacher.
func ServeAttach(w http.ResponseWriter, req *http.Request, attacher Attacher, podName string, uid types.UID, container string, streamOpts *Options, idleTimeout, streamCreationTimeout time.Duration, supportedProtocols []string) {
	ctx, ok := createStreams(req, w, streamOpts, supportedProtocols, idleTimeout, streamCreationTimeout)
	if !ok {
		// error is handled by createStreams
		return
	}
	defer ctx.conn.Close()

	err := attacher.AttachContainer(podName, uid, container, ctx.stdinStream, ctx.stdoutStream, ctx.stderrStream, ctx.tty, ctx.resizeChan)
	if err != nil {
		err = fmt.Errorf("error attaching to container: %v", err)
		runtime.HandleError(err)
		ctx.writeStatus(apierrors.NewInternalError(err))
	} else {
		ctx.writeStatus(&apierrors.StatusError{ErrStatus: api.Status{
			Status: api.StatusSuccess,
		}})
	}
}
