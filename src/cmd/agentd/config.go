package main

import (
	"time"

	envstruct "code.cloudfoundry.org/go-envstruct"
)

// Config holds the configuration for the system metrics agent.
type Config struct {
	SampleInterval time.Duration `env:"SAMPLE_INTERVAL,            report"`
	Deployment     string        `env:"DEPLOYMENT, report"`
	Job            string        `env:"JOB, report"`
	Index          string        `env:"INDEX, report"`
	IP             string        `env:"IP, report"`

	DebugPort      uint16 `env:"DEBUG_PORT, report"`
	MetricPort     uint16 `env:"METRIC_PORT, report, required"`
	LimitedMetrics bool   `env:"LIMITED_METRICS, report, required"`

	CACertPath string `env:"CA_CERT_PATH, required, report"`
	CertPath   string `env:"CERT_PATH, required, report"`
	KeyPath    string `env:"KEY_PATH, required, report"`
}

func loadConfig() (Config, error) {
	cfg := Config{
		SampleInterval: time.Second * 15,
		MetricPort:     0,
	}

	if err := envstruct.Load(&cfg); err != nil {
		return cfg, err
	}

	envstruct.ReportWriter = stdLogger.Writer()
	if err := envstruct.WriteReport(&cfg); err != nil {
		errLogger.Printf("failed to write config to stderr: %s\n", err)
	}

	return cfg, nil
}
