package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	_ "net/http/pprof"

	"code.cloudfoundry.org/go-envstruct"
	"code.cloudfoundry.org/go-loggregator/metrics"
	"code.cloudfoundry.org/system-metrics/cmd/leadership-election/app"
)

func main() {
	log.Printf("Starting Leadership Election...")
	defer log.Printf("Closing Leadership Election...")

	cfg, err := app.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

	envstruct.WriteReport(&cfg)

	logger := log.New(os.Stderr, "", log.LstdFlags)
	m := metrics.NewRegistry(
		logger,
		metrics.WithTLSServer(
			int(cfg.MetricsServer.Port),
			cfg.MetricsServer.CertFile,
			cfg.MetricsServer.KeyFile,
			cfg.MetricsServer.CAFile,
		),
	)

	a := app.New(
		cfg.NodeIndex,
		cfg.NodeAddrs,
		app.WithLogger(logger),
		app.WithMetrics(m),
		app.WithPort(int(cfg.Port)),
	)

	a.Start(
		cfg.CAFile,
		cfg.CertFile,
		cfg.KeyFile,
	)

	// health endpoints (pprof and expvar)
	log.Printf("Health: %s", http.ListenAndServe(fmt.Sprintf("localhost:%d", cfg.HealthPort), nil))
}
