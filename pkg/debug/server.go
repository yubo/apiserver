package debug

import (
	"context"
	"expvar"
	"fmt"
	"net"
	"net/http"
	"net/http/pprof"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"k8s.io/klog/v2"
)

const (
	shutDownTimeout        = 5 * time.Second
	defaultKeepAlivePeriod = 3 * time.Minute
)

type server struct {
	address string
	handler http.Handler
}

func newServer(cf *config) (*server, error) {
	mux := http.NewServeMux()

	if cf.Metrics {
		mux.Handle(cf.MetricsPath, promhttp.Handler())
	}
	if cf.Expvar {
		mux.Handle("/debug/vars", expvar.Handler())
	}

	if cf.Pprof {
		mux.HandleFunc("/debug/pprof", redirectTo("/debug/pprof/"))
		mux.HandleFunc("/debug/pprof/", pprof.Index)
		mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
		//mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	}

	return &server{
		address: *cf.Address,
		handler: mux,
	}, nil
}

func (s *server) start(ctx context.Context) (<-chan struct{}, error) {
	listener, err := net.Listen("tcp", s.address)
	if err != nil {
		return nil, err
	}

	server := &http.Server{
		Addr:    listener.Addr().String(),
		Handler: s.handler,
	}

	// Shutdown server gracefully.
	stoppedCh := make(chan struct{})
	go func() {
		<-ctx.Done()

		ctx2, cancel := context.WithTimeout(context.Background(), shutDownTimeout)
		server.Shutdown(ctx2)
		cancel()
		close(stoppedCh)
	}()

	go func() {
		err := server.Serve(tcpKeepAliveListener{listener})

		msg := fmt.Sprintf("Stopped listening on %s", listener.Addr().String())
		select {
		case <-ctx.Done():
			klog.Info(msg)
		default:
			panic(fmt.Sprintf("%s due to error: %v", msg, err))
		}
	}()

	return stoppedCh, nil
}

// tcpKeepAliveListener sets TCP keep-alive timeouts on accepted
// connections. It's used by ListenAndServe and ListenAndServeTLS so
// dead TCP connections (e.g. closing laptop mid-download) eventually
// go away.
//
// Copied from Go 1.7.2 net/http/server.go
type tcpKeepAliveListener struct {
	net.Listener
}

func (ln tcpKeepAliveListener) Accept() (net.Conn, error) {
	c, err := ln.Listener.Accept()
	if err != nil {
		return nil, err
	}
	if tc, ok := c.(*net.TCPConn); ok {
		tc.SetKeepAlive(true)
		tc.SetKeepAlivePeriod(defaultKeepAlivePeriod)
	}
	return c, nil
}

// redirectTo redirects request to a certain destination.
func redirectTo(to string) func(http.ResponseWriter, *http.Request) {
	return func(rw http.ResponseWriter, req *http.Request) {
		http.Redirect(rw, req, to, http.StatusFound)
	}
}
