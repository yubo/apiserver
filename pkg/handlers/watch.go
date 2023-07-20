package handlers

import (
	"bytes"
	"fmt"
	"net/http"
	"time"

	"github.com/yubo/apiserver/pkg/handlers/negotiation"
	"github.com/yubo/apiserver/pkg/request"
	"github.com/yubo/apiserver/pkg/scheme"
	"github.com/yubo/apiserver/pkg/server/httplog"
	"github.com/yubo/golib/api"
	"github.com/yubo/golib/api/errors"
	"github.com/yubo/golib/runtime"
	"github.com/yubo/golib/runtime/serializer/streaming"
	"github.com/yubo/golib/stream/wsstream"
	utilruntime "github.com/yubo/golib/util/runtime"
	"github.com/yubo/golib/watch"
	"golang.org/x/net/websocket"
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

	// negotiate for the stream serializer from the scope's serializer
	serializer, err := negotiation.NegotiateOutputMediaTypeStream(req, codecs)
	if err != nil {
		return err
	}
	framer := serializer.StreamSerializer.Framer
	streamSerializer := serializer.StreamSerializer.Serializer
	//encoder := codecs.EncoderForVersion(streamSerializer)
	useTextFraming := serializer.EncodesAsText
	if framer == nil {
		return fmt.Errorf("no framer defined for %q available for embedded encoding", serializer.MediaType)
	}
	// TODO: next step, get back mediaTypeOptions from negotiate and return the exact value here
	mediaType := serializer.MediaType
	if mediaType != runtime.ContentTypeJSON {
		mediaType += ";stream=watch"
	}

	// locate the appropriate embedded encoder based on the transform
	//var embeddedEncoder runtime.Encoder
	//contentKind, contentSerializer, transform := targetEncodingForTransform(scope, mediaTypeOptions, req)
	//if transform {
	//	info, ok := runtime.SerializerInfoForMediaType(contentSerializer.SupportedMediaTypes(), serializer.MediaType)
	//	if !ok {
	//		scope.err(fmt.Errorf("no encoder for %q exists in the requested target %#v", serializer.MediaType, contentSerializer), w, req)
	//		return
	//	}
	//	embeddedEncoder = contentSerializer.EncoderForVersion(info.Serializer, contentKind.GroupVersion())
	//} else {
	//	embeddedEncoder = scope.Serializer.EncoderForVersion(serializer.Serializer, contentKind.GroupVersion())
	//}

	var serverShuttingDownCh <-chan struct{}
	if signals := request.ServerShutdownSignalFrom(req.Context()); signals != nil {
		serverShuttingDownCh = signals.ShuttingDown()
	}

	server := &WatchServer{
		Watching: watcher,

		UseTextFraming: useTextFraming,
		MediaType:      mediaType,
		Framer:         framer,
		Encoder:        streamSerializer,
		//EmbeddedEncoder: embeddedEncoder,

		//Fixup: func(obj runtime.Object) runtime.Object {
		//	result, err := transformObject(ctx, obj, options, mediaTypeOptions, scope, req)
		//	if err != nil {
		//		utilruntime.HandleError(fmt.Errorf("failed to transform object %v: %v", reflect.TypeOf(obj), err))
		//		return obj
		//	}
		//	// When we are transformed to a table, use the table options as the state for whether we
		//	// should print headers - on watch, we only want to print table headers on the first object
		//	// and omit them on subsequent events.
		//	if tableOptions, ok := options.(*metav1.TableOptions); ok {
		//		tableOptions.NoHeaders = true
		//	}
		//	return result
		//},

		TimeoutFactory:       &realTimeoutFactory{timeout},
		ServerShuttingDownCh: serverShuttingDownCh,
	}

	return server.ServeHTTP(w, req)
}

// WatchServer serves a watch.Interface over a websocket or vanilla HTTP.
type WatchServer struct {
	Watching watch.Interface
	//Scope    *RequestScope

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
	// used to correct the object before we send it to the serializer
	//Fixup func(runtime.Object) runtime.Object

	TimeoutFactory       TimeoutFactory
	ServerShuttingDownCh <-chan struct{}
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
		utilruntime.HandleError(err)
		return errors.NewInternalError(err)
	}

	framer := s.Framer.NewFrameWriter(w)
	if framer == nil {
		// programmer error
		err := fmt.Errorf("no stream framing support is available for media type %q", s.MediaType)
		utilruntime.HandleError(err)
		return errors.NewBadRequest(err.Error())
	}

	e := streaming.NewEncoder(framer, s.Encoder)
	var memoryAllocator runtime.MemoryAllocator

	if encoder, supportsAllocator := s.Encoder.(runtime.EncoderWithAllocator); supportsAllocator {
		memoryAllocator = runtime.AllocatorPool.Get().(*runtime.Allocator)
		defer runtime.AllocatorPool.Put(memoryAllocator)
		e = streaming.NewEncoderWithAllocator(framer, encoder, memoryAllocator)
	} else {
		e = streaming.NewEncoder(framer, s.Encoder)
	}

	// ensure the connection times out
	timeoutCh, cleanup := s.TimeoutFactory.TimeoutCh()
	defer cleanup()

	// begin the stream
	w.Header().Set("Content-Type", s.MediaType)
	w.Header().Set("Transfer-Encoding", "chunked")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	internalEvent := &api.InternalEvent{}
	outEvent := &api.WatchEvent{}
	ch := s.Watching.ResultChan()
	done := req.Context().Done()

	//embeddedEncodeFn := s.EmbeddedEncoder.Encode
	//if encoder, supportsAllocator := s.EmbeddedEncoder.(runtime.EncoderWithAllocator); supportsAllocator {
	//	if memoryAllocator == nil {
	//		// don't put the allocator inside the embeddedEncodeFn as that would allocate memory on every call.
	//		// instead, we allocate the buffer for the entire watch session and release it when we close the connection.
	//		memoryAllocator = runtime.AllocatorPool.Get().(*runtime.Allocator)
	//		defer runtime.AllocatorPool.Put(memoryAllocator)
	//	}
	//	embeddedEncodeFn = func(obj runtime.Object, w io.Writer) error {
	//		return encoder.EncodeWithAllocator(obj, w, memoryAllocator)
	//	}
	//}

	for {
		select {
		case <-s.ServerShuttingDownCh:
			// the server has signaled that it is shutting down (not accepting
			// any new request), all active watch request(s) should return
			// immediately here. The WithWatchTerminationDuringShutdown server
			// filter will ensure that the response to the client is rate
			// limited in order to avoid any thundering herd issue when the
			// client(s) try to reestablish the WATCH on the other
			// available apiserver instance(s).
			return nil
		case <-done:
			return nil
		case <-timeoutCh:
			return nil
		case event, ok := <-ch:
			if !ok {
				// End of results.
				return nil
			}
			//metrics.WatchEvents.WithContext(req.Context()).WithLabelValues(kind.Group, kind.Version, kind.Kind).Inc()

			*outEvent = api.WatchEvent{}
			// create the external type directly and encode it.  Clients will only recognize the serialization we provide.
			// The internal event is being reused, not reallocated so its just a few extra assignments to do it this way
			// and we get the benefit of using conversion functions which already have to stay in sync
			*internalEvent = api.InternalEvent(event)
			err := api.Convert_v1_InternalEvent_To_v1_WatchEvent(internalEvent, outEvent)
			if err != nil {
				utilruntime.HandleError(fmt.Errorf("unable to convert watch object: %v", err))
				// client disconnect.
				return nil
			}

			if err := e.Encode(outEvent); err != nil {
				utilruntime.HandleError(fmt.Errorf("unable to encode watch object %T: %v (%#v)", outEvent, err, e))
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

	internalEvent := &api.InternalEvent{}
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

			// the internal event will be versioned by the encoder
			// create the external type directly and encode it.  Clients will only recognize the serialization we provide.
			// The internal event is being reused, not reallocated so its just a few extra assignments to do it this way
			// and we get the benefit of using conversion functions which already have to stay in sync
			outEvent := &api.WatchEvent{}
			*internalEvent = api.InternalEvent(event)
			err := api.Convert_v1_InternalEvent_To_v1_WatchEvent(internalEvent, outEvent)
			if err != nil {
				utilruntime.HandleError(fmt.Errorf("unable to convert watch object: %v", err))
				// client disconnect.
				return
			}
			if err := s.Encoder.Encode(outEvent, streamBuf); err != nil {
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
