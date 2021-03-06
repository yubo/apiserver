package filters

import (
	"fmt"
	"net/http"

	"github.com/emicklei/go-restful/v3"
	"github.com/yubo/apiserver/pkg/request"
	"github.com/yubo/apiserver/pkg/responsewriters"
	"github.com/yubo/apiserver/pkg/session/types"
	"k8s.io/klog/v2"
)

// http filter
func WithSession(handler http.Handler, sm types.SessionManager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		defer klog.V(8).Infof("leaving filters.WithSession")

		s, err := sm.Start(w, req)
		if err != nil {
			responsewriters.InternalError(w, req, err)
			return
		}
		defer s.Update(w)

		req = req.WithContext(request.WithSession(req.Context(), s))

		handler.ServeHTTP(w, req)
	})
}

// go-restful filter
func Session(sm types.SessionManager) restful.FilterFunction {
	return func(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
		s, err := sm.Start(resp, req.Request)
		if err != nil {
			responsewriters.InternalError(resp, req.Request, fmt.Errorf("session start err %s", err))
			return
		}
		defer s.Update(resp)

		ctx := request.WithSession(req.Request.Context(), s)
		req.Request = req.Request.WithContext(ctx)

		chain.ProcessFilter(req, resp)
	}
}
