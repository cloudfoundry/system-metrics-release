package debugserver

import (
	"fmt"
	"net/http"
	"net/http/pprof"
	"time"
)

type DebugServer struct {
	srv *http.Server
}

func New(port uint16) *DebugServer {
	m := http.NewServeMux()
	m.HandleFunc("/debug/pprof/", pprof.Index)
	m.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	m.HandleFunc("/debug/pprof/profile", pprof.Profile)
	m.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	m.HandleFunc("/debug/pprof/trace", pprof.Trace)
	return &DebugServer{
		srv: &http.Server{
			Addr:              fmt.Sprintf("127.0.0.1:%d", port),
			ReadHeaderTimeout: 5 * time.Second,
			Handler:           m,
		},
	}
}

func (s *DebugServer) Start() error {
	return s.srv.ListenAndServe()
}

func (s *DebugServer) Stop() error {
	return s.srv.Close()
}

func (s *DebugServer) Addr() string {
	return s.srv.Addr
}
