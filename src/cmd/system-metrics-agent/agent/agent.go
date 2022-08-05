package agent

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"code.cloudfoundry.org/system-metrics/pkg/collector"
	"code.cloudfoundry.org/system-metrics/pkg/egress/stats"
	"code.cloudfoundry.org/tlsconfig"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const statOrigin = "system_metrics_agent"

type Agent struct {
	cfg           Config
	log           *log.Logger
	metricsLis    net.Listener
	metricsServer http.Server
	mu            sync.Mutex
	inputFunc     collector.InputFunc
}

func New(i collector.InputFunc, cfg Config, log *log.Logger) *Agent {
	return &Agent{
		cfg:       cfg,
		log:       log,
		inputFunc: i,
	}
}

func (a *Agent) Run() {
	metricsURL := fmt.Sprintf(":%d", a.cfg.MetricPort)
	a.startMetricsServer(metricsURL)
}

func (a *Agent) MetricsAddr() string {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.metricsLis == nil {
		return ""
	}

	return a.metricsLis.Addr().String()
}

func (a *Agent) Shutdown(ctx context.Context) {
	a.mu.Lock()
	defer a.mu.Unlock()

	err := a.metricsServer.Shutdown(ctx)
	if err != nil {
		a.log.Printf("failed to shutdown: %s\n", err)
	}
}

func (a *Agent) startMetricsServer(addr string) {
	labels := map[string]string{
		"source_id":  statOrigin,
		"deployment": a.cfg.Deployment,
		"job":        a.cfg.Job,
		"index":      a.cfg.Index,
		"ip":         a.cfg.IP,
	}

	promRegisterer := prometheus.NewRegistry()
	promRegistry := stats.NewPromRegistry(promRegisterer)
	promSender := stats.NewPromSender(promRegistry, statOrigin, a.cfg.LimitedMetrics, labels)

	router := http.NewServeMux()
	router.Handle("/metrics", promhttp.HandlerFor(promRegisterer, promhttp.HandlerOpts{}))

	a.setup(addr, router)

	go collector.NewProcessor(
		a.inputFunc,
		[]collector.StatsSender{promSender},
		a.cfg.SampleInterval,
		a.log,
	).Run()

	log.Printf("Metrics server closing: %s", a.metricsServer.ServeTLS(a.metricsLis, "", ""))
}

func (a *Agent) setup(addr string, router *http.ServeMux) {
	a.mu.Lock()
	defer a.mu.Unlock()

	tlsConfig, err := tlsconfig.Build(
		tlsconfig.WithInternalServiceDefaults(),
		tlsconfig.WithIdentityFromFile(a.cfg.CertPath, a.cfg.KeyPath),
	).Server(
		tlsconfig.WithClientAuthenticationFromFile(a.cfg.CACertPath),
	)

	if err != nil {
		log.Fatalf("Unable to setup tls for metrics endpoint (%s): %s", addr, err)
	}

	a.metricsServer = http.Server{
		Addr:              addr,
		ReadTimeout:       5 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      5 * time.Second,
		Handler:           router,
		TLSConfig:         tlsConfig,
	}

	a.metricsLis, err = net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("Unable to setup metrics endpoint (%s): %s", addr, err)
	}
	log.Printf("Metrics endpoint is listening on %s", a.metricsLis.Addr().String())
}
