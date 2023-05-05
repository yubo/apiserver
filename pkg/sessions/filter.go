package sessions

import (
	"net/http"

	"github.com/emicklei/go-restful/v3"
	"k8s.io/klog/v2"
)

var (
	defaultStore Store
)

func SetStore(s Store) {
	if defaultStore != nil {
		klog.Warningf("store %s(%s) has already set, use %s(%s) instead",
			defaultStore.Name(), defaultStore.Type(),
			s.Name(), s.Type(),
		)
	}
	defaultStore = s
}

// http filter
func WithSessions(handler http.Handler) http.Handler {
	if defaultStore == nil {
		return handler
	}
	return Sessions(handler, defaultStore.Name(), defaultStore)
}

// go-restful filter
func SessionsFilter() restful.FilterFunction {
	if defaultStore == nil {
		return nil
	}

	return func(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
		s := &session{"default", req.Request, defaultStore, nil, false, resp}

		ctx := withSession(req.Request.Context(), s)
		req.Request = req.Request.WithContext(ctx)

		chain.ProcessFilter(req, resp)
	}
}

func Sessions(handler http.Handler, name string, store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		s := &session{name, req, store, nil, false, w}
		req = req.WithContext(withSession(req.Context(), s))

		handler.ServeHTTP(w, req)
	}
}

func SessionsMany(handler http.Handler, names []string, store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		sessions := make(map[string]Session, len(names))
		for _, name := range names {
			sessions[name] = &session{name, req, store, nil, false, w}
		}

		req = req.WithContext(withManySession(req.Context(), sessions))
		handler.ServeHTTP(w, req)
	}
}
