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

package responsewriters

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	//"k8s.io/apiserver/pkg/features"

	"github.com/yubo/apiserver/pkg/audit"
	"github.com/yubo/apiserver/pkg/handlers/negotiation"
	"github.com/yubo/apiserver/pkg/metrics"
	"github.com/yubo/apiserver/pkg/request"
	"github.com/yubo/golib/runtime"
	"github.com/yubo/golib/stream/wsstream"
	"github.com/yubo/golib/util/flushwriter"
	utilruntime "github.com/yubo/golib/util/runtime"

	//utilfeature "github.com/yubo/golib/util/feature"
	utiltrace "github.com/yubo/golib/util/trace"
)

// StreamObject performs input stream negotiation from a ResourceStreamer and writes that to the response.
// If the client requests a websocket upgrade, negotiate for a websocket reader protocol (because many
// browser clients cannot easily handle binary streaming protocols).
func StreamObject(statusCode int, s runtime.NegotiatedSerializer, stream ResourceStreamer, w http.ResponseWriter, req *http.Request) {
	out, flush, contentType, err := stream.InputStream(req.Context(), req.Header.Get("Accept"))
	if err != nil {
		ErrorNegotiated(err, s, w, req)
		return
	}
	if out == nil {
		// No output provided - return StatusNoContent
		w.WriteHeader(http.StatusNoContent)
		return
	}
	defer out.Close()

	if wsstream.IsWebSocketRequest(req) {
		r := wsstream.NewReader(out, true, wsstream.NewDefaultReaderProtocols())
		if err := r.Copy(w, req); err != nil {
			utilruntime.HandleError(fmt.Errorf("error encountered while streaming results via websocket: %v", err))
		}
		return
	}

	if len(contentType) == 0 {
		contentType = "application/octet-stream"
	}
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(statusCode)
	// Flush headers, if possible
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
	writer := w.(io.Writer)
	if flush {
		writer = flushwriter.Wrap(w)
	}
	io.Copy(writer, out)
}

// SerializeObject renders an object in the content type negotiated by the client using the provided encoder.
// The context is optional and can be nil. This method will perform optional content compression if requested by
// a client and the feature gate for APIResponseCompression is enabled.
func SerializeObject(mediaType string, encoder runtime.Encoder, hw http.ResponseWriter, req *http.Request, statusCode int, object runtime.Object) {
	trace := utiltrace.New("SerializeObject",
		utiltrace.Field{Key: "method", Value: req.Method},
		utiltrace.Field{Key: "url", Value: req.URL.Path},
		utiltrace.Field{Key: "protocol", Value: req.Proto},
		utiltrace.Field{Key: "mediaType", Value: mediaType},
	)
	defer trace.LogIfLong(5 * time.Second)

	w := &deferredResponseWriter{
		mediaType:       mediaType,
		statusCode:      statusCode,
		contentEncoding: negotiateContentEncoding(req),
		hw:              hw,
		trace:           trace,
	}

	err := encoder.Encode(object, w)
	if err == nil {
		err = w.Close()
		if err != nil {
			// we cannot write an error to the writer anymore as the Encode call was successful.
			utilruntime.HandleError(fmt.Errorf("apiserver was unable to close cleanly the response writer: %v", err))
		}
		return
	}

	// make a best effort to write the object if a failure is detected
	utilruntime.HandleError(fmt.Errorf("apiserver was unable to write a JSON response: %v", err))
	status := ErrorToAPIStatus(err)
	candidateStatusCode := int(status.Code)
	// if the current status code is successful, allow the error's status code to overwrite it
	if statusCode >= http.StatusOK && statusCode < http.StatusBadRequest {
		w.statusCode = candidateStatusCode
	}
	output, err := runtime.Encode(encoder, status)
	if err != nil {
		w.mediaType = "text/plain"
		output = []byte(fmt.Sprintf("%s: %s", status.Reason, status.Message))
	}
	if _, err := w.Write(output); err != nil {
		utilruntime.HandleError(fmt.Errorf("apiserver was unable to write a fallback JSON response: %v", err))
	}
	w.Close()
}

var gzipPool = &sync.Pool{
	New: func() interface{} {
		gw, err := gzip.NewWriterLevel(nil, defaultGzipContentEncodingLevel)
		if err != nil {
			panic(err)
		}
		return gw
	},
}

const (
	// defaultGzipContentEncodingLevel is set to 4 which uses less CPU than the default level
	defaultGzipContentEncodingLevel = 4
	// defaultGzipThresholdBytes is compared to the size of the first write from the stream
	// (usually the entire object), and if the size is smaller no gzipping will be performed
	// if the client requests it.
	defaultGzipThresholdBytes = 128 * 1024
)

// negotiateContentEncoding returns a supported client-requested content encoding for the
// provided request. It will return the empty string if no supported content encoding was
// found or if response compression is disabled.
func negotiateContentEncoding(req *http.Request) string {
	encoding := req.Header.Get("Accept-Encoding")
	if len(encoding) == 0 {
		return ""
	}
	//if !utilfeature.DefaultFeatureGate.Enabled(features.APIResponseCompression) {
	//	return ""
	//}
	for len(encoding) > 0 {
		var token string
		if next := strings.Index(encoding, ","); next != -1 {
			token = encoding[:next]
			encoding = encoding[next+1:]
		} else {
			token = encoding
			encoding = ""
		}
		switch strings.TrimSpace(token) {
		case "gzip":
			return "gzip"
		}
	}
	return ""
}

type deferredResponseWriter struct {
	mediaType       string
	statusCode      int
	contentEncoding string

	hasWritten bool
	hw         http.ResponseWriter
	w          io.Writer

	trace *utiltrace.Trace
}

func (w *deferredResponseWriter) Write(p []byte) (n int, err error) {
	if w.trace != nil {
		// This Step usually wraps in-memory object serialization.
		w.trace.Step("About to start writing response", utiltrace.Field{Key: "size", Value: len(p)})

		firstWrite := !w.hasWritten
		defer func() {
			w.trace.Step("Write call finished",
				utiltrace.Field{Key: "writer", Value: fmt.Sprintf("%T", w.w)},
				utiltrace.Field{Key: "size", Value: len(p)},
				utiltrace.Field{Key: "firstWrite", Value: firstWrite})
		}()
	}
	if w.hasWritten {
		return w.w.Write(p)
	}
	w.hasWritten = true

	hw := w.hw
	header := hw.Header()
	switch {
	case w.contentEncoding == "gzip" && len(p) > defaultGzipThresholdBytes:
		header.Set("Content-Encoding", "gzip")
		header.Add("Vary", "Accept-Encoding")

		gw := gzipPool.Get().(*gzip.Writer)
		gw.Reset(hw)

		w.w = gw
	default:
		w.w = hw
	}

	header.Set("Content-Type", w.mediaType)
	hw.WriteHeader(w.statusCode)
	return w.w.Write(p)
}

func (w *deferredResponseWriter) Close() error {
	if !w.hasWritten {
		return nil
	}
	var err error
	switch t := w.w.(type) {
	case *gzip.Writer:
		err = t.Close()
		t.Reset(nil)
		gzipPool.Put(t)
	}
	return err
}

var nopCloser = ioutil.NopCloser(nil)

// WriteObject renders an object in the content type negotiated by the client.
func WriteObject(w http.ResponseWriter, req *http.Request, statusCode int, object interface{}) {
	WriteRawJSON(int(statusCode), object, w)
	return

}

// ResourceStreamer is an interface implemented by objects that prefer to be streamed from the server
// instead of decoded directly.
type ResourceStreamer interface {
	// InputStream should return an io.ReadCloser if the provided object supports streaming. The desired
	// api version and an accept header (may be empty) are passed to the call. If no error occurs,
	// the caller may return a flag indicating whether the result should be flushed as writes occur
	// and a content type string that indicates the type of the stream.
	// If a null stream is returned, a StatusNoContent response wil be generated.
	InputStream(ctx context.Context, acceptHeader string) (stream io.ReadCloser, flush bool, mimeType string, err error)
}

// WriteObjectNegotiated renders an object in the content type negotiated by the client.
func WriteObjectNegotiated(s runtime.NegotiatedSerializer, w http.ResponseWriter, req *http.Request, statusCode int, object runtime.Object) {
	stream, ok := object.(ResourceStreamer)
	if ok {
		requestInfo, _ := request.RequestInfoFrom(req.Context())
		metrics.RecordLongRunning(req, requestInfo, metrics.APIServerComponent, func() {
			StreamObject(statusCode, s, stream, w, req)
		})
		return
	}

	_, serializer, err := negotiation.NegotiateOutputMediaType(req, s)
	if err != nil {
		// if original statusCode was not successful we need to return the original error
		// we cannot hide it behind negotiation problems
		if statusCode < http.StatusOK || statusCode >= http.StatusBadRequest {
			WriteRawJSON(int(statusCode), object, w)
			return
		}
		status := ErrorToAPIStatus(err)
		WriteRawJSON(int(status.Code), status, w)
		return
	}

	if ae := request.AuditEventFrom(req.Context()); ae != nil {
		audit.LogResponseObject(ae, object)
	}

	encoder := s.EncoderForVersion(serializer.Serializer)
	SerializeObject(serializer.MediaType, encoder, w, req, statusCode, object)
}

// ErrorNegotiated renders an error to the response. Returns the HTTP status code of the error.
// The context is optional and may be nil.
func ErrorNegotiated(err error, s runtime.NegotiatedSerializer, w http.ResponseWriter, req *http.Request) int {
	status := ErrorToAPIStatus(err)
	code := int(status.Code)
	// when writing an error, check to see if the status indicates a retry after period
	if status.Details != nil && status.Details.RetryAfterSeconds > 0 {
		delay := strconv.Itoa(int(status.Details.RetryAfterSeconds))
		w.Header().Set("Retry-After", delay)
	}

	if code == http.StatusNoContent {
		w.WriteHeader(code)
		return code
	}

	WriteObjectNegotiated(s, w, req, code, status)
	return code
}

// WriteRawJSON writes a non-API object in JSON.
func WriteRawJSON(statusCode int, object interface{}, w http.ResponseWriter) {
	output, err := json.Marshal(object)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	w.Write(output)
}

func HttpRedirect(w http.ResponseWriter, url string) {
	w.Header().Add("location", url)
	w.WriteHeader(http.StatusFound)
}
