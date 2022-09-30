package app_test

import (
	"log"
	"net"
	"net/http"
	"testing"
	"time"

	"code.cloudfoundry.org/tlsconfig"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestApp(t *testing.T) {
	log.SetOutput(GinkgoWriter)
	RegisterFailHandler(Fail)
	RunSpecs(t, "System Metrics Agent App Suite")
}

func newTLSClient(cert, key, ca, commonName string) *http.Client {
	tlsConfig, err := tlsconfig.Build(
		tlsconfig.WithInternalServiceDefaults(),
		tlsconfig.WithIdentityFromFile(cert, key),
	).Client(
		tlsconfig.WithAuthorityFromFile(ca),
		tlsconfig.WithServerName(commonName),
	)

	if err != nil {
		log.Panicf("failed to load API client certificates: %s", err)
	}

	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig:       tlsConfig,
	}

	return &http.Client{
		Transport: transport,
	}
}
