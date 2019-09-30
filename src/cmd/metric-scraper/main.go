package main

import (
	metrics "code.cloudfoundry.org/go-metric-registry"
	"log"
	"os"

	"code.cloudfoundry.org/system-metrics/cmd/metric-scraper/app"
)

func main() {
	log := log.New(os.Stderr, "", log.LstdFlags)
	log.Printf("starting Metrics Scraper...")
	defer log.Printf("closing Metrics Scraper...")

	cfg := app.LoadConfig(log)

	metricClient := metrics.NewRegistry(
		log,
		metrics.WithTLSServer(
			int(cfg.MetricsServer.Port),
			cfg.MetricsServer.CertFile,
			cfg.MetricsServer.KeyFile,
			cfg.MetricsServer.CAFile,
		),
	)

	app.NewMetricScraper(cfg, log, metricClient).Run()
}
