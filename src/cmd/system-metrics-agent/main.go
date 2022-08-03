package main

import (
	"log"
	"os"

	"code.cloudfoundry.org/system-metrics/cmd/system-metrics-agent/app"
	"code.cloudfoundry.org/system-metrics/pkg/collector"
)

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

	cfg, err := app.LoadConfig(errLogger)
	if err != nil {
		errLogger.Fatalf("failed to load config from environment: %s\n", err)
	}

	c := collector.New(errLogger)
	app.NewSystemMetricsAgent(c.Collect, cfg, errLogger).Run()
}
