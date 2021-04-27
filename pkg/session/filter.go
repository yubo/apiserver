package session

import (
	"net/http"

	"github.com/yubo/apiserver/pkg/request"
	"github.com/yubo/apiserver/pkg/responsewriters"
	"github.com/yubo/golib/net/session"
	"k8s.io/klog/v2"
)

func WithSession(handler http.Handler, sm session.SessionManager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		klog.V(8).Infof("entering session filter")

		s, err := sm.Start(w, req)
		if err != nil {
			responsewriters.InternalError(w, req, err)
			return
		}

		req = req.WithContext(request.WithSession(req.Context(), s))

		klog.V(8).Infof("leaving authn filter")
		handler.ServeHTTP(w, req)
	})
}
