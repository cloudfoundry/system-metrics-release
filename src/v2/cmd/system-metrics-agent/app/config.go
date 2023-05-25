package app

import (
	"time"

	envstruct "code.cloudfoundry.org/go-envstruct"
)

// Config holds the configuration for the system metrics agent server.
type Config struct {
	LogLevel string `env:"LOG_LEVEL, report"`

	Interval time.Duration `env:"INTERVAL, report"`

	Port     uint16 `env:"PORT, report"`
	CAFile   string `env:"CA_FILE, report"`
	CertFile string `env:"CERT_FILE, report"`
	KeyFile  string `env:"KEY_FILE, report"`

	Deployment string `env:"DEPLOYMENT, report"`
	Job        string `env:"JOB, report"`
	Index      string `env:"INDEX, report"`
	IP         string `env:"IP, report"`
}

// NewConfig returns a new Config.
func NewConfig() *Config {
	return &Config{
		Interval: 15 * time.Second,
		LogLevel: "info",
	}
}

// Load loads the configuration from the environment.
func (cfg *Config) Load() error {
	return envstruct.Load(cfg)
}
