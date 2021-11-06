package routes

import (
	"expvar"

	"github.com/yubo/apiserver/pkg/server/mux"
)

// Profiling adds handlers for pprof under /debug/vars.
type Expvar struct{}

// Install adds the expvar webservice to the given mux.
func (e Expvar) Install(c *mux.PathRecorderMux) {
	c.UnlistedHandle("/debug/vars", expvar.Handler())
}
