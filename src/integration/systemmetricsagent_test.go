package integration_test

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"time"

	"code.cloudfoundry.org/system-metrics/internal/testhelper"
	"code.cloudfoundry.org/system-metrics/pkg/plumbing"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("System Metrics Agent", func() {
	const (
		testDeployment = "test-deployment"
		testJob        = "test-job"
		testIndex      = "test-index"
		testIp         = "test-ip"
	)

	var (
		port    int
		client  *http.Client
		session *gexec.Session
	)

	BeforeEach(func() {
		port = 8080 + GinkgoParallelProcess()

		certpaths := testhelper.GenerateCerts("system-metrics-agent-ca")
		caPath := certpaths.CA()
		certPath := certpaths.Cert("system-metrics-agent")
		keyPath := certpaths.Key("system-metrics-agent")

		client = plumbing.NewTLSHTTPClient(
			certPath,
			keyPath,
			caPath,
			"system-metrics-agent",
		)

		os.Setenv("CA_CERT_PATH", caPath)
		os.Setenv("CERT_PATH", certPath)
		os.Setenv("KEY_PATH", keyPath)
		os.Setenv("LIMITED_METRICS", "false")
		os.Setenv("METRIC_PORT", fmt.Sprintf("%d", port))
		os.Setenv("DEPLOYMENT", testDeployment)
		os.Setenv("JOB", testJob)
		os.Setenv("INDEX", testIndex)
		os.Setenv("IP", testIp)

		cmd := exec.Command(agentPath)
		var err error
		session, err = gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		session.Kill()
	})

	It("listens on a local port with TLS", func() {
		url := fmt.Sprintf("https://localhost:%d/metrics", port)
		Eventually(func(g Gomega) {
			resp, err := client.Get(url)
			g.Expect(err).To(BeNil())
			g.Expect(resp.StatusCode).To(Equal(http.StatusOK))
		}).Should(Succeed())
	})

	It("emits all the metrics", func() {
		// Not all metrics appear to be generated right away, some seem to
		// require more time to appear. Hence the Eventually assertion.
		url := fmt.Sprintf("https://localhost:%d/metrics", port)
		Eventually(func(g Gomega) {
			resp, err := client.Get(url)
			g.Expect(err).To(BeNil())
			g.Expect(resp.StatusCode).To(Equal(http.StatusOK))
			b, err := io.ReadAll(resp.Body)
			g.Expect(err).To(BeNil())
			body := string(b)
			g.Expect(len(body)).ToNot(Equal(0))

			var metrics []string
			switch runtime.GOOS {
			case "linux":
				metrics = []string{
					"system_cpu_core_idle",
					"system_cpu_core_sys",
					"system_cpu_core_user",
					"system_cpu_core_wait",
					"system_cpu_idle",
					"system_cpu_sys",
					"system_cpu_user",
					"system_cpu_wait",
					"system_disk_ephemeral_inode_percent",
					"system_disk_ephemeral_io_time",
					"system_disk_ephemeral_percent",
					"system_disk_ephemeral_read_bytes",
					"system_disk_ephemeral_read_time",
					"system_disk_ephemeral_write_bytes",
					"system_disk_ephemeral_write_time",
					"system_disk_persistent_inode_percent",
					"system_disk_persistent_io_time",
					"system_disk_persistent_percent",
					"system_disk_persistent_read_bytes",
					"system_disk_persistent_read_time",
					"system_disk_persistent_write_bytes",
					"system_disk_persistent_write_time",
					"system_disk_system_inode_percent",
					"system_disk_system_io_time",
					"system_disk_system_percent",
					"system_disk_system_read_bytes",
					"system_disk_system_read_time",
					"system_disk_system_write_bytes",
					"system_disk_system_write_time",
					"system_healthy",
					"system_load_15m",
					"system_load_1m",
					"system_load_5m",
					"system_mem_kb",
					"system_mem_percent",
					"system_network_bytes_received",
					"system_network_bytes_sent",
					"system_network_drop_in",
					"system_network_drop_out",
					"system_network_error_in",
					"system_network_error_out",
					"system_network_ip_forwarding",
					"system_network_packets_received",
					"system_network_packets_sent",
					"system_network_tcp_active_opens",
					"system_network_tcp_curr_estab",
					"system_network_tcp_retrans_segs",
					"system_network_udp_in_errors",
					"system_network_udp_lite_in_errors",
					"system_network_udp_no_ports",
					"system_swap_kb",
					"system_swap_percent",
				}
			case "windows":
				metrics = []string{
					"system_cpu_core_idle",
					"system_cpu_core_sys",
					"system_cpu_core_user",
					"system_cpu_core_wait",
					"system_cpu_idle",
					"system_cpu_sys",
					"system_cpu_user",
					"system_cpu_wait",
					"system_disk_ephemeral_inode_percent",
					"system_disk_ephemeral_io_time",
					"system_disk_ephemeral_percent",
					"system_disk_ephemeral_read_bytes",
					"system_disk_ephemeral_read_time",
					"system_disk_ephemeral_write_bytes",
					"system_disk_ephemeral_write_time",
					"system_disk_persistent_inode_percent",
					"system_disk_persistent_io_time",
					"system_disk_persistent_percent",
					"system_disk_persistent_read_bytes",
					"system_disk_persistent_read_time",
					"system_disk_persistent_write_bytes",
					"system_disk_persistent_write_time",
					"system_disk_system_inode_percent",
					"system_disk_system_io_time",
					"system_disk_system_percent",
					"system_disk_system_read_bytes",
					"system_disk_system_read_time",
					"system_disk_system_write_bytes",
					"system_disk_system_write_time",
					"system_healthy",
					"system_mem_kb",
					"system_mem_percent",
					"system_network_bytes_received",
					"system_network_bytes_sent",
					"system_network_drop_in",
					"system_network_drop_out",
					"system_network_error_in",
					"system_network_error_out",
					"system_network_packets_received",
					"system_network_packets_sent",
					"system_swap_kb",
					"system_swap_percent",
				}
			default:
				metrics = []string{
					"system_cpu_core_idle",
					"system_cpu_core_sys",
					"system_cpu_core_user",
					"system_cpu_core_wait",
					"system_cpu_idle",
					"system_cpu_sys",
					"system_cpu_user",
					"system_cpu_wait",
					"system_load_1m",
					"system_load_5m",
					"system_mem_kb",
					"system_mem_percent",
					"system_swap_kb",
					"system_swap_percent",
				}
			}
			for _, m := range metrics {
				g.Expect(body).To(ContainSubstring(m))
			}
		}, 1*time.Minute, 1*time.Second).Should(Succeed())
	})

	It("uses correct labels", func() {
		// Metrics aren't emitted right away, they need some time to start
		// appearing in responses. Hence the Eventually assertion
		url := fmt.Sprintf("https://localhost:%d/metrics", port)
		Eventually(func(g Gomega) {
			resp, err := client.Get(url)
			g.Expect(err).To(BeNil())
			g.Expect(resp.StatusCode).To(Equal(http.StatusOK))
			b, err := io.ReadAll(resp.Body)
			g.Expect(err).To(BeNil())
			body := string(b)
			g.Expect(len(body)).ToNot(Equal(0))

			g.Expect(body).To(ContainSubstring(testDeployment))
			g.Expect(body).To(ContainSubstring(testJob))
			g.Expect(body).To(ContainSubstring(testIndex))
			g.Expect(body).To(ContainSubstring(testIp))
		}, 1*time.Minute, 1*time.Second).Should(Succeed())
	})
})
