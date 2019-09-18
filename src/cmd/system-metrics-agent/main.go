package main

import (
	"log"
	"os"

	"code.cloudfoundry.org/system-metrics/pkg/collector"

	"code.cloudfoundry.org/system-metrics/cmd/system-metrics-agent/app"
)

func main() {
	log := log.New(os.Stderr, "", log.LstdFlags)
	log.Println("starting system-metrics-agent")
	defer log.Println("stopping system-metrics-agent")

	cfg := app.LoadConfig()

	c := collector.New(log)
	app.NewSystemMetricsAgent(c.Collect, cfg, log).Run()
}
