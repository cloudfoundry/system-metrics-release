package main

import (
	"log"
	"os"
	"time"

	"code.cloudfoundry.org/system-metrics/pkg/collector"
	"code.cloudfoundry.org/system-metrics/pkg/debugserver"
	"code.cloudfoundry.org/system-metrics/pkg/egress/stats"
	"code.cloudfoundry.org/system-metrics/pkg/metricserver"

	"code.cloudfoundry.org/go-envstruct"
	"code.cloudfoundry.org/tlsconfig"
	"github.com/prometheus/client_golang/prometheus"
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

	go func() {
		err := debugserver.New(cfg.DebugPort).Start()
		if err != nil {
			errLogger.Println(err)
		}
	}()

	c := collector.New(errLogger)
	reg := prometheus.NewRegistry()
	sender := stats.NewPromSender(stats.NewPromRegistry(reg), "", cfg.LimitedMetrics, labels("", &cfg))
	go collector.NewProcessor(c.Collect, []collector.StatsSender{sender}, cfg.SampleInterval, errLogger).Run()

	tlsCfg, err := tlsconfig.Build(
		tlsconfig.WithInternalServiceDefaults(),
		tlsconfig.WithIdentityFromFile(cfg.CertPath, cfg.KeyPath),
	).Server(
		tlsconfig.WithClientAuthenticationFromFile(cfg.CACertPath),
	)
	if err != nil {
		errLogger.Fatalf("failed to build tls config: %s\n", err)
	}
	errLogger.Fatalf("metric server error: %s\n", metricserver.New(cfg.MetricPort, tlsCfg, reg).Run())
}

func labels(sourceID string, cfg *config) map[string]string {
	return map[string]string{
		"source_id":  sourceID,
		"deployment": cfg.Deployment,
		"job":        cfg.Job,
		"index":      cfg.Index,
		"ip":         cfg.IP,
	}
}
