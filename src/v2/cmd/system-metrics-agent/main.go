// The System Metrics Agent is responsible for recording system metrics on VMs
// and exposing them via a prometheus endpoint.
package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	"code.cloudfoundry.org/go-envstruct"
	"code.cloudfoundry.org/system-metrics-release/src/v2/cmd/system-metrics-agent/app"
)

func main() {
	l := log.New(os.Stderr, "", 0)

	cfg := app.NewConfig()
	if err := cfg.Load(); err != nil {
		l.Fatal(err)
	}
	envstruct.WriteReport(cfg) //nolint:errcheck

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)

	errCh := make(chan error)
	app := app.NewServer(cfg)
	go func() {
		errCh <- app.Run(context.Background())
	}()

	select {
	case <-sigCh:
		l.Println("\nshut down by signal")
		return
	case err := <-errCh:
		if err != nil {
			l.Fatal(err)
		}
	}
}
