package agent_test

import (
	"context"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	"code.cloudfoundry.org/system-metrics/pkg/collector"

	"code.cloudfoundry.org/system-metrics/cmd/system-metrics-agent/agent"
	"code.cloudfoundry.org/system-metrics/internal/testhelper"
	"code.cloudfoundry.org/system-metrics/pkg/plumbing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("SystemMetricsAgent", func() {
	var (
		app       *agent.Agent
		testCerts = testhelper.GenerateCerts("systemMetricsCA")
	)

	BeforeEach(func() {
		inputFunc := func() (collector.SystemStat, error) {
			return defaultStat, nil
		}

		app = agent.New(
			inputFunc,
			agent.Config{
				SampleInterval: time.Millisecond,
				Deployment:     "some-deployment",
				Job:            "some-job",
				Index:          "some-index",
				IP:             "some-ip",
				CACertPath:     testCerts.CA(),
				CertPath:       testCerts.Cert("system-metrics-agent"),
				KeyPath:        testCerts.Key("system-metrics-agent"),
				LimitedMetrics: true,
			},
			log.New(GinkgoWriter, "", log.LstdFlags),
		)
	})

	It("has an http listener for PProf", func() {
		go app.Run()
		defer app.Shutdown(context.Background())

		var addr string
		Eventually(func() int {
			addr = app.DebugAddr()
			return len(addr)
		}).ShouldNot(Equal(0))

		resp, err := http.Get("http://" + addr + "/debug/pprof/")
		Expect(err).ToNot(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
	})

	It("has a prom exposition endpoint", func() {
		go app.Run()
		defer app.Shutdown(context.Background())

		var addr string
		Eventually(func() int {
			addr = app.MetricsAddr()
			return len(addr)
		}).ShouldNot(Equal(0))

		client := plumbing.NewTLSHTTPClient(
			testCerts.Cert("system-metrics-agent"),
			testCerts.Key("system-metrics-agent"),
			testCerts.CA(),
			"system-metrics-agent",
		)
		resp, err := client.Get("https://" + addr + "/metrics")
		Expect(err).ToNot(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
	})

	It("emits correct metrics", func() {
		inputFunc := func() (collector.SystemStat, error) {
			return defaultStat, nil
		}
		app = agent.New(
			inputFunc,
			agent.Config{
				SampleInterval: time.Millisecond,
				Deployment:     "some-deployment",
				Job:            "some-job",
				Index:          "some-index",
				IP:             "some-ip",
				CACertPath:     testCerts.CA(),
				CertPath:       testCerts.Cert("system-metrics-agent"),
				KeyPath:        testCerts.Key("system-metrics-agent"),
			},
			log.New(GinkgoWriter, "", log.LstdFlags),
		)
		go app.Run()
		defer app.Shutdown(context.Background())

		var addr string
		Eventually(func() int {
			addr = app.MetricsAddr()
			return len(addr)
		}).ShouldNot(Equal(0))

		client := plumbing.NewTLSHTTPClient(
			testCerts.Cert("system-metrics-agent"),
			testCerts.Key("system-metrics-agent"),
			testCerts.CA(),
			"system-metrics-agent",
		)
		resp, err := client.Get("https://" + addr + "/metrics")
		Expect(err).ToNot(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
		body, err := io.ReadAll(resp.Body)
		Expect(err).To(BeNil())
		Expect(strings.Count(string(body), "\n")).To(Equal(120))
	})

	It("limits metrics emitted", func() {
		inputFunc := func() (collector.SystemStat, error) {
			return defaultStat, nil
		}
		app = agent.New(
			inputFunc,
			agent.Config{
				SampleInterval: time.Millisecond,
				Deployment:     "some-deployment",
				Job:            "some-job",
				Index:          "some-index",
				IP:             "some-ip",
				CACertPath:     testCerts.CA(),
				CertPath:       testCerts.Cert("system-metrics-agent"),
				KeyPath:        testCerts.Key("system-metrics-agent"),
				LimitedMetrics: true,
			},
			log.New(GinkgoWriter, "", log.LstdFlags),
		)
		go app.Run()
		defer app.Shutdown(context.Background())

		var addr string
		Eventually(func() int {
			addr = app.MetricsAddr()
			return len(addr)
		}).ShouldNot(Equal(0))

		client := plumbing.NewTLSHTTPClient(
			testCerts.Cert("system-metrics-agent"),
			testCerts.Key("system-metrics-agent"),
			testCerts.CA(),
			"system-metrics-agent",
		)
		resp, err := client.Get("https://" + addr + "/metrics")
		Expect(err).ToNot(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
		body, err := io.ReadAll(resp.Body)
		Expect(err).To(BeNil())
		Expect(strings.Count(string(body), "\n")).To(Equal(45))
	})

	It("contains default prom labels", func() {
		go app.Run()
		defer app.Shutdown(context.Background())

		var addr string
		Eventually(func() int {
			addr = app.MetricsAddr()
			return len(addr)
		}).ShouldNot(Equal(0))

		Eventually(hasDefaultLabels(addr, testCerts)).Should(BeTrue())
	})
})

func hasDefaultLabels(addr string, testCerts *testhelper.TestCerts) func() bool {
	return func() bool {
		client := plumbing.NewTLSHTTPClient(
			testCerts.Cert("system-metrics-agent"),
			testCerts.Key("system-metrics-agent"),
			testCerts.CA(),
			"system-metrics-agent",
		)
		resp, err := client.Get("https://" + addr + "/metrics")
		Expect(err).ToNot(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusOK))

		body, err := ioutil.ReadAll(resp.Body)
		Expect(err).ToNot(HaveOccurred())

		if len(body) > 0 {
			bodyStr := string(body)
			Expect(bodyStr).To(ContainSubstring(`origin="system_metrics_agent"`))
			Expect(bodyStr).To(ContainSubstring(`source_id="system_metrics_agent"`))
			Expect(bodyStr).To(ContainSubstring(`deployment="some-deployment"`))
			Expect(bodyStr).To(ContainSubstring(`job="some-job"`))
			Expect(bodyStr).To(ContainSubstring(`index="some-index"`))
			Expect(bodyStr).To(ContainSubstring(`ip="some-ip"`))

			return true
		}

		return false
	}
}

var (
	defaultStat = collector.SystemStat{
		MemKB:      1025,
		MemPercent: 10.01,

		SwapKB:      2049,
		SwapPercent: 20.01,

		Load1M:  1.1,
		Load5M:  5.5,
		Load15M: 15.15,

		CPUStat: collector.CPUStat{
			User:   25.25,
			System: 52.52,
			Idle:   10.10,
			Wait:   22.22,
		},

		SystemDisk: collector.DiskStat{
			Present: true,

			Percent:      35.0,
			InodePercent: 45.0,

			ReadBytes:  10,
			WriteBytes: 20,
			ReadTime:   30,
			WriteTime:  40,
			IOTime:     50,
		},

		EphemeralDisk: collector.DiskStat{
			Present: true,

			Percent:      55.0,
			InodePercent: 65.0,

			ReadBytes:  100,
			WriteBytes: 200,
			ReadTime:   300,
			WriteTime:  400,
			IOTime:     500,
		},

		PersistentDisk: collector.DiskStat{
			Present: true,

			Percent:      75.0,
			InodePercent: 85.0,

			ReadBytes:  1000,
			WriteBytes: 2000,
			ReadTime:   3000,
			WriteTime:  4000,
			IOTime:     5000,
		},

		ProtoCounters: collector.ProtoCountersStat{
			Present:         true,
			IPForwarding:    1,
			UDPNoPorts:      2,
			UDPInErrors:     3,
			UDPLiteInErrors: 4,
			TCPActiveOpens:  5,
			TCPCurrEstab:    6,
			TCPRetransSegs:  7,
		},

		Health: collector.HealthStat{
			Present: true,
			Healthy: true,
		},
	}
)
