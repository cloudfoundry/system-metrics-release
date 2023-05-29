package integration_test

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"

	sma "code.cloudfoundry.org/system-metrics-release/src/integration/systemmetricsagent"
	"code.cloudfoundry.org/system-metrics-release/src/internal/testhelper"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("System Metrics Agent", Ordered, func() {
	var (
		tc *testhelper.TestCerts

		session *gexec.Session
	)

	BeforeAll(func() {
		// build agent
		DeferCleanup(gexec.CleanupBuildArtifacts)
		pathToSystemMetricsAgent, err := gexec.Build("code.cloudfoundry.org/system-metrics-release/src/cmd/system-metrics-agent")
		Expect(err).NotTo(HaveOccurred())

		// setup agent configuration
		DeferCleanup(os.Setenv, "METRIC_PORT", os.Getenv("METRIC_PORT"))
		os.Setenv("METRIC_PORT", "8080")
		DeferCleanup(os.Setenv, "LIMITED_METRICS", os.Getenv("LIMITED_METRICS"))
		os.Setenv("LIMITED_METRICS", "false")
		DeferCleanup(os.Setenv, "SAMPLE_INTERVAL", os.Getenv("SAMPLE_INTERVAL"))
		os.Setenv("SAMPLE_INTERVAL", "1s")
		DeferCleanup(os.Setenv, "DEPLOYMENT", os.Getenv("DEPLOYMENT"))
		os.Setenv("DEPLOYMENT", "test-deployment")
		DeferCleanup(os.Setenv, "JOB", os.Getenv("JOB"))
		os.Setenv("JOB", "test-job")
		DeferCleanup(os.Setenv, "INDEX", os.Getenv("INDEX"))
		os.Setenv("INDEX", "test-index")
		DeferCleanup(os.Setenv, "IP", os.Getenv("IP"))
		os.Setenv("IP", "test-ip")
		tc = testhelper.GenerateCerts("systemMetricsCA")
		DeferCleanup(os.Setenv, "CA_CERT_PATH", os.Getenv("CA_CERT_PATH"))
		os.Setenv("CA_CERT_PATH", tc.CA())
		DeferCleanup(os.Setenv, "CERT_PATH", os.Getenv("CERT_PATH"))
		os.Setenv("CERT_PATH", tc.Cert("system-metrics-agent"))
		DeferCleanup(os.Setenv, "KEY_PATH", os.Getenv("KEY_PATH"))
		os.Setenv("KEY_PATH", tc.Key("system-metrics-agent"))

		// run agent
		command := exec.Command(pathToSystemMetricsAgent)
		session, err = gexec.Start(command, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterAll(func() {
		session.Terminate().Wait()
	})

	Describe("metrics endpoint", func() {
		It("rejects HTTP requests", func() {
			var resp *http.Response
			Eventually(func() error {
				var err error
				resp, err = http.Get("http://localhost:8080/metrics")
				return err
			}, "3s").Should(Succeed())
			defer resp.Body.Close()

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
			client, err := sma.NewClient(tc)
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() error {
				var err error
				resp, err := client.Get("https://localhost:8080/metrics")
				if err != nil {
					return err
				}
				defer resp.Body.Close()

				if resp.StatusCode != 200 {
					return fmt.Errorf("expected 200 status code, got %d", resp.StatusCode)
				}

				b, err := io.ReadAll(resp.Body)
				if err != nil {
					return fmt.Errorf("failed to read response body: %w", err)
				}

				body := string(b)
				if !strings.Contains(body, "# HELP") {
					return fmt.Errorf("unexpected response body: %s", body)
				}
				return nil
			}, "5s").Should(Succeed())
		})

		Context("when running in Windows", func() {
			BeforeEach(func() {
				if runtime.GOOS != "windows" {
					Skip("Skipping on non-Windows")
				}
			})

			It("returns expected metrics", func() {
				client, err := sma.NewClient(tc)
				Expect(err).NotTo(HaveOccurred())

				var body string
				Eventually(func() error {
					var err error
					resp, err := client.Get("https://localhost:8080/metrics")
					if err != nil {
						return err
					}
					defer resp.Body.Close()

					if resp.StatusCode != 200 {
						return fmt.Errorf("expected 200 status code, got %d", resp.StatusCode)
					}

					b, err := io.ReadAll(resp.Body)
					if err != nil {
						return fmt.Errorf("failed to read response body: %w", err)
					}
					body = string(b)
					return nil
				}, "3s").Should(Succeed())

				fmt.Println(body)

				for _, m := range sma.WINDOWS_METRIC_NAMES {
					Expect(body).To(MatchRegexp(fmt.Sprintf(sma.MetricSubstrForRegexCmp, m)))
				}
			})
		})

		Context("when running in Linux", func() {
			BeforeEach(func() {
				if runtime.GOOS != "linux" {
					Skip("Skipping on non-Linux")
				}
			})

			It("returns expected metrics", func() {
				client, err := sma.NewClient(tc)
				Expect(err).NotTo(HaveOccurred())

				var body string
				Eventually(func() error {
					var err error
					resp, err := client.Get("https://localhost:8080/metrics")
					if err != nil {
						return err
					}
					defer resp.Body.Close()

					if resp.StatusCode != 200 {
						return fmt.Errorf("expected 200 status code, got %d", resp.StatusCode)
					}

					b, err := io.ReadAll(resp.Body)
					if err != nil {
						return fmt.Errorf("failed to read response body: %w", err)
					}
					body = string(b)
					return nil
				}, "3s").Should(Succeed())

				fmt.Println(body)

				for _, m := range sma.LINUX_METRIC_NAMES {
					Expect(body).To(MatchRegexp(fmt.Sprintf(sma.MetricSubstrForRegexCmp, m)))
				}
			})
		})
	})
})
