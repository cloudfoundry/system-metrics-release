package main

import (
	"log"
	"os"
	"time"

	"code.cloudfoundry.org/go-envstruct"
	"code.cloudfoundry.org/system-metrics/cmd/system-metrics-agent/agent"
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

type config struct {
	SampleInterval time.Duration `env:"SAMPLE_INTERVAL, report"`
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

func main() {
	stdLogger.Println("starting system-metrics-agent")
	defer stdLogger.Println("stopping system-metrics-agent")

	cfg := config{
		SampleInterval: time.Second * 15,
		MetricPort:     0,
	}
	if err := envstruct.Load(&cfg); err != nil {
		errLogger.Fatalf("failed to load config from environment: %s\n", err)
	}
	envstruct.ReportWriter = os.Stdout
	if err := envstruct.WriteReport(&cfg); err != nil {
		errLogger.Printf("failed to write config to stderr: %s\n", err)
	}

	c := collector.New(errLogger)
	agent.New(c.Collect, agent.Config(cfg), errLogger).Run()
}
