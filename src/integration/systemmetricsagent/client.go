// Package systemmetricsagent provides some helper functions, structs and
// constants for integration testing system metrics agent.
package systemmetricsagent

import (
	"net/http"

	"code.cloudfoundry.org/system-metrics-release/src/internal/testhelper"
	"code.cloudfoundry.org/tlsconfig"
)

func NewClient(tc *testhelper.TestCerts) (*http.Client, error) {
	cfg, err := tlsconfig.Build(
		tlsconfig.WithInternalServiceDefaults(),
		tlsconfig.WithIdentityFromFile(tc.Cert("system-metrics-agent"), tc.Key("system-metrics-agent")),
	).Client(
		tlsconfig.WithAuthorityFromFile(tc.CA()),
		tlsconfig.WithServerName("system-metrics-agent"),
	)
	if err != nil {
		return nil, err
	}

	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: cfg,
		},
	}, nil
}
