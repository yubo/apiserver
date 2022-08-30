package filters

import (
	"net/http"
	"net/http/httputil"

	"github.com/emicklei/go-restful/v3"
	"github.com/yubo/apiserver/pkg/request"
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
			b, err := httputil.DumpRequest(req, true)
			klog.InfoS("httpdump", "req", string(b), "err", err)
		}

		handler.ServeHTTP(w, req)
	})
}
