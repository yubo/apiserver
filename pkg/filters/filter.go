package filters

import (
	"net/http"
	"net/http/httputil"

	"github.com/emicklei/go-restful/v3"
	"github.com/yubo/apiserver/pkg/request"
	httplib "github.com/yubo/golib/net/http"
	"github.com/yubo/golib/stream/wsstream"
	"k8s.io/klog/v2"
)

func HttpFilter(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
	ctx := req.Request.Context()
	ctx = request.WithResp(ctx, resp)

	req.Request = req.Request.WithContext(ctx)

	chain.ProcessFilter(req, resp)
}

// WithHSTS is a simple HSTS implementation that wraps an http Handler.
// If hstsDirectives is empty or nil, no HSTS support is installed.
func WithHttpDump(handler http.Handler) http.Handler {
	if !klog.V(10).Enabled() {
		return handler
	}

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if !wsstream.IsWebSocketRequest(req) {
			// req
			b1, e1 := httputil.DumpRequest(req, true)
			recorder := httplib.NewResponseWriterRecorder(w, req)
			w = recorder

			defer func() {
				b2, e2 := recorder.Dump(true)
				klog.InfoS("httpdump", "req", string(b1), "req_err", e1, "resp", string(b2), "resp_err", e2)
			}()
		}

		handler.ServeHTTP(w, req)
	})
}
