package metricserver

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type MetricServer struct {
	srv *http.Server
}

func New(port uint16, tlsCfg *tls.Config, gatherer prometheus.Gatherer) *MetricServer {
	m := http.NewServeMux()
	m.Handle("/metrics", promhttp.HandlerFor(gatherer, promhttp.HandlerOpts{}))
	return &MetricServer{
		srv: &http.Server{
			Addr:              fmt.Sprintf(":%d", port),
			ReadTimeout:       5 * time.Second,
			ReadHeaderTimeout: 5 * time.Second,
			WriteTimeout:      5 * time.Second,
			Handler:           m,
			TLSConfig:         tlsCfg,
		},
	}
}

func (s *MetricServer) Run() error {
	return s.srv.ListenAndServeTLS("", "")
}

func (s *MetricServer) Stop() error {
	return s.srv.Close()
}

func (s *MetricServer) Addr() string {
	return s.srv.Addr
}
