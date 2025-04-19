package integration_test

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"code.cloudfoundry.org/system-metrics-release/src/internal/testhelper"
	"code.cloudfoundry.org/tlsconfig"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("System Metrics Agent", Ordered, func() {
	var (
		tc *testhelper.TestCerts
	)

	BeforeAll(func() {
		// build agent
		DeferCleanup(gexec.CleanupBuildArtifacts)
		pathToSystemMetricsAgent, err := gexec.Build("code.cloudfoundry.org/system-metrics-release/src/cmd/system-metrics-agent")
		Expect(err).NotTo(HaveOccurred())

		// setup agent configuration
		DeferCleanup(os.Setenv, "METRIC_PORT", os.Getenv("METRIC_PORT"))
		err = os.Setenv("METRIC_PORT", "8080")
		Expect(err).NotTo(HaveOccurred())

		DeferCleanup(os.Setenv, "LIMITED_METRICS", os.Getenv("LIMITED_METRICS"))
		err = os.Setenv("LIMITED_METRICS", "false")
		Expect(err).NotTo(HaveOccurred())

		DeferCleanup(os.Setenv, "SAMPLE_INTERVAL", os.Getenv("SAMPLE_INTERVAL"))
		err = os.Setenv("SAMPLE_INTERVAL", "1s")
		Expect(err).NotTo(HaveOccurred())

		DeferCleanup(os.Setenv, "DEPLOYMENT", os.Getenv("DEPLOYMENT"))
		err = os.Setenv("DEPLOYMENT", "test-deployment")
		Expect(err).NotTo(HaveOccurred())

		DeferCleanup(os.Setenv, "JOB", os.Getenv("JOB"))
		err = os.Setenv("JOB", "test-job")
		Expect(err).NotTo(HaveOccurred())

		DeferCleanup(os.Setenv, "INDEX", os.Getenv("INDEX"))
		err = os.Setenv("INDEX", "test-index")
		Expect(err).NotTo(HaveOccurred())

		DeferCleanup(os.Setenv, "IP", os.Getenv("IP"))
		err = os.Setenv("IP", "test-ip")
		Expect(err).NotTo(HaveOccurred())

		tc = testhelper.GenerateCerts("systemMetricsCA")

		DeferCleanup(os.Setenv, "CA_CERT_PATH", os.Getenv("CA_CERT_PATH"))
		err = os.Setenv("CA_CERT_PATH", tc.CA())
		Expect(err).NotTo(HaveOccurred())

		DeferCleanup(os.Setenv, "CERT_PATH", os.Getenv("CERT_PATH"))
		err = os.Setenv("CERT_PATH", tc.Cert("system-metrics-agent"))
		Expect(err).NotTo(HaveOccurred())

		DeferCleanup(os.Setenv, "KEY_PATH", os.Getenv("KEY_PATH"))
		err = os.Setenv("KEY_PATH", tc.Key("system-metrics-agent"))
		Expect(err).NotTo(HaveOccurred())

		// run agent
		command := exec.Command(pathToSystemMetricsAgent)
		_, err = gexec.Start(command, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterAll(func() {
		gexec.KillAndWait()
	})

	Describe("metrics endpoint", func() {
		It("rejects HTTP requests", func() {
			var resp *http.Response
			Eventually(func() error {
				var err error
				resp, err = http.Get("http://localhost:8080/metrics")
				return err
			}, "3s").Should(Succeed())
			defer resp.Body.Close() //nolint:errcheck

			Expect(resp.StatusCode).To(Equal(400))
		})

		It("rejects HTTPS requests without proper mTLS", func() {
			client := &http.Client{
				Transport: &http.Transport{
					TLSClientConfig: &tls.Config{
						InsecureSkipVerify: true,
					},
				},
			}

			Consistently(func() error {
				_, err := client.Get("https://localhost:8080/metrics")
				return err
			}, "3s").ShouldNot(Succeed())
		})

		It("returns a successful prometheus response", func() {
			cfg, err := tlsconfig.Build(
				tlsconfig.WithInternalServiceDefaults(),
				tlsconfig.WithIdentityFromFile(tc.Cert("system-metrics-agent"), tc.Key("system-metrics-agent")),
			).Client(
				tlsconfig.WithAuthorityFromFile(tc.CA()),
				tlsconfig.WithServerName("system-metrics-agent"),
			)
			Expect(err).NotTo(HaveOccurred())

			client := &http.Client{
				Transport: &http.Transport{
					TLSClientConfig: cfg,
				},
			}

			expectedSubstr := `
# HELP system_cpu_idle vm metric
# TYPE system_cpu_idle gauge
system_cpu_idle{deployment="test-deployment",index="test-index",ip="test-ip",job="test-job",origin="system_metrics_agent",source_id="system_metrics_agent",unit="Percent"}`

			var resp *http.Response
			Eventually(func() error {
				var err error
				resp, err = client.Get("https://localhost:8080/metrics")
				if err != nil {
					return err
				}
				defer resp.Body.Close() //nolint:errcheck

				if resp.StatusCode != 200 {
					return fmt.Errorf("expected 200 status code, got %d", resp.StatusCode)
				}

				b, err := io.ReadAll(resp.Body)
				if err != nil {
					return fmt.Errorf("failed to read response body: %w", err)
				}

				body := string(b)
				if !strings.Contains(body, expectedSubstr) {
					return fmt.Errorf("unexpected response body: %s", body)
				}

				return nil
			}, "5s").Should(Succeed())
		})
	})
})
