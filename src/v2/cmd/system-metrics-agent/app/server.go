package app

import (
	"context"
	"fmt"
	"net/http"
)

// Server collects metrics about the system and exposes them via a prometheus
// endpoint.
type Server struct {
	cfg *Config
}

// NewServer returns a new App.
func NewServer(cfg *Config) *Server {
	return &Server{
		cfg: cfg,
	}
}

// Run starts collecting metrics and listening for prometheus requests.
func (s *Server) Run(ctx context.Context) error {
	http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok")) //nolint:errcheck
	})
	return http.ListenAndServe(fmt.Sprintf(":%d", s.cfg.Port), nil)
}
