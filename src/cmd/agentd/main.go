package main

import (
	"log"
	"os"

	"code.cloudfoundry.org/system-metrics/pkg/collector"
	"code.cloudfoundry.org/system-metrics/pkg/debugserver"
	"code.cloudfoundry.org/system-metrics/pkg/egress/stats"
	"code.cloudfoundry.org/system-metrics/pkg/metricserver"
	"code.cloudfoundry.org/tlsconfig"
	"github.com/prometheus/client_golang/prometheus"
)

const statOrigin = "system_metrics_agent"

var (
	stdLogger *log.Logger
	errLogger *log.Logger
)

func init() {
	stdLogger = log.New(os.Stdout, "", log.LstdFlags)
	errLogger = log.New(os.Stderr, "", log.LstdFlags)
}

func main() {
	stdLogger.Println("starting system-metrics-agent")
	defer stdLogger.Println("stopping system-metrics-agent")

	cfg, err := loadConfig()
	if err != nil {
		errLogger.Fatalf("failed to load config from environment: %s\n", err)
	}

	go func() {
		err := debugserver.New(cfg.DebugPort).Start()
		if err != nil {
			errLogger.Println(err)
		}
	}()

	c := collector.New(errLogger)

	labels := map[string]string{
		"source_id":  statOrigin,
		"deployment": cfg.Deployment,
		"job":        cfg.Job,
		"index":      cfg.Index,
		"ip":         cfg.IP,
	}
	reg := prometheus.NewRegistry()
	statsReg := stats.NewPromRegistry(reg)
	sender := stats.NewPromSender(statsReg, statOrigin, cfg.LimitedMetrics, labels)
	tlsConfig, err := tlsconfig.Build(
		tlsconfig.WithInternalServiceDefaults(),
		tlsconfig.WithIdentityFromFile(cfg.CertPath, cfg.KeyPath),
	).Server(
		tlsconfig.WithClientAuthenticationFromFile(cfg.CACertPath),
	)
	if err != nil {
		errLogger.Fatalf("unable to build TLS config for metrics server: %s\n", err)
	}

	srv := metricserver.New(cfg.MetricPort, reg, tlsConfig)

	go collector.NewProcessor(
		c.Collect,
		[]collector.StatsSender{sender},
		cfg.SampleInterval,
		errLogger,
	).Run()

	srv.Run()
}
