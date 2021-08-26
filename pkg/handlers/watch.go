package handlers

import (
	"bytes"
	"fmt"
	"net/http"
	"time"

	"github.com/yubo/apiserver/pkg/apiserver/httplog"
	"github.com/yubo/apiserver/pkg/handlers/negotiation"
	"github.com/yubo/apiserver/pkg/watch"
	"github.com/yubo/golib/api/errors"
	"github.com/yubo/golib/runtime"
	"github.com/yubo/golib/runtime/serializer/streaming"
	"github.com/yubo/golib/scheme"
	utilruntime "github.com/yubo/golib/util/runtime"
	"github.com/yubo/golib/util/wsstream"
	"golang.org/x/net/websocket"
	"k8s.io/klog/v2"
)

// nothing will ever be sent down this channel
var neverExitWatch <-chan time.Time = make(chan time.Time)

// timeoutFactory abstracts watch timeout logic for testing
type TimeoutFactory interface {
	TimeoutCh() (<-chan time.Time, func() bool)
}

// realTimeoutFactory implements timeoutFactory
type realTimeoutFactory struct {
	timeout time.Duration
}

// TimeoutCh returns a channel which will receive something when the watch times out,
// and a cleanup function to call when this happens.
func (w *realTimeoutFactory) TimeoutCh() (<-chan time.Time, func() bool) {
	if w.timeout == 0 {
		return neverExitWatch, func() bool { return false }
	}
	t := time.NewTimer(w.timeout)
	return t.C, t.Stop
}

// serveWatch will serve a watch response.
// TODO: the functionality in this method and in WatchServer.Serve is not cleanly decoupled.
func ServeWatch(watcher watch.Interface, req *http.Request, w http.ResponseWriter, timeout time.Duration) error {
	defer watcher.Stop()

	codecs := scheme.Codecs

	serializer, err := negotiation.NegotiateOutputMediaTypeStream(req, codecs)
	if err != nil {
		return err
	}
	framer := serializer.StreamSerializer.Framer
	streamSerializer := serializer.StreamSerializer.Serializer
	encoder := codecs.EncoderForVersion(streamSerializer)
	useTextFraming := serializer.EncodesAsText
	if framer == nil {
		return fmt.Errorf("no framer defined for %q available for embedded encoding", serializer.MediaType)
	}
	// TODO: next step, get back mediaTypeOptions from negotiate and return the exact value here
	mediaType := serializer.MediaType
	if mediaType != runtime.ContentTypeJSON {
		mediaType += ";stream=watch"
	}

	//embeddedEncoder := codecs.EncoderForVersion(serializer.Serializer)

	server := &WatchServer{
		Watching: watcher,

		UseTextFraming: useTextFraming,
		MediaType:      mediaType,
		Framer:         framer,
		Encoder:        encoder,
		//EmbeddedEncoder: embeddedEncoder,

		TimeoutFactory: &realTimeoutFactory{timeout},
	}

	return server.ServeHTTP(w, req)
}

// WatchServer serves a watch.Interface over a websocket or vanilla HTTP.
type WatchServer struct {
	Watching watch.Interface

	// true if websocket messages should use text framing (as opposed to binary framing)
	UseTextFraming bool
	// the media type this watch is being served with
	MediaType string
	// used to frame the watch stream
	Framer runtime.Framer
	// used to encode the watch stream event itself
	Encoder runtime.Encoder
	// used to encode the nested object in the watch stream
	//EmbeddedEncoder runtime.Encoder

	TimeoutFactory TimeoutFactory
}

// ServeHTTP serves a series of encoded events via HTTP with Transfer-Encoding: chunked
// or over a websocket connection.
func (s *WatchServer) ServeHTTP(w http.ResponseWriter, req *http.Request) error {
	//metrics.RegisteredWatchers.WithLabelValues().Inc()
	//defer metrics.RegisteredWatchers.WithLabelValues().Dec()

	w = httplog.Unlogged(req, w)

	if wsstream.IsWebSocketRequest(req) {
		w.Header().Set("Content-Type", s.MediaType)
		websocket.Handler(s.HandleWS).ServeHTTP(w, req)
		return nil
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		err := fmt.Errorf("unable to start watch - can't get http.Flusher: %#v", w)
		return errors.NewInternalError(err)
	}

	framer := s.Framer.NewFrameWriter(w)
	if framer == nil {
		// programmer error
		return errors.NewBadRequest(fmt.Sprintf("no stream framing support is available for media type %q", s.MediaType))
	}

	e := streaming.NewEncoder(framer, s.Encoder)

	// ensure the connection times out
	timeoutCh, cleanup := s.TimeoutFactory.TimeoutCh()
	defer cleanup()

	// begin the stream
	w.Header().Set("Content-Type", s.MediaType)
	w.Header().Set("Transfer-Encoding", "chunked")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	ch := s.Watching.ResultChan()
	done := req.Context().Done()

	for {
		select {
		case <-done:
			return nil
		case <-timeoutCh:
			return nil
		case event, ok := <-ch:
			if !ok {
				// End of results.
				return nil
			}

			if err := e.Encode(event); err != nil {
				klog.Error(fmt.Errorf("unable to encode watch object %T: %v (%#v)", event, err, e))
				// client disconnect.
				return nil
			}
			if len(ch) == 0 {
				flusher.Flush()
			}
		}
	}
}

// HandleWS implements a websocket handler.
func (s *WatchServer) HandleWS(ws *websocket.Conn) {
	defer ws.Close()
	done := make(chan struct{})

	go func() {
		defer utilruntime.HandleCrash()
		// This blocks until the connection is closed.
		// Client should not send anything.
		wsstream.IgnoreReceives(ws, 0)
		// Once the client closes, we should also close
		close(done)
	}()

	streamBuf := &bytes.Buffer{}
	ch := s.Watching.ResultChan()

	for {
		select {
		case <-done:
			return
		case event, ok := <-ch:
			if !ok {
				// End of results.
				return
			}
			if err := s.Encoder.Encode(event, streamBuf); err != nil {
				// encoding error
				utilruntime.HandleError(fmt.Errorf("unable to encode event: %v", err))
				return
			}
			if s.UseTextFraming {
				if err := websocket.Message.Send(ws, streamBuf.String()); err != nil {
					// Client disconnect.
					return
				}
			} else {
				if err := websocket.Message.Send(ws, streamBuf.Bytes()); err != nil {
					// Client disconnect.
					return
				}
			}
			streamBuf.Reset()
		}
	}
}
