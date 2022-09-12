package metricserver_test

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"code.cloudfoundry.org/system-metrics/internal/testhelper"
	"code.cloudfoundry.org/system-metrics/pkg/metricserver"
	"code.cloudfoundry.org/system-metrics/pkg/plumbing"
	"code.cloudfoundry.org/tlsconfig"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus"
)

var _ = Describe("Metric Server", func() {
	var (
		port      uint16
		s         *metricserver.MetricServer
		client    *http.Client
		gaugeName = "test_gauge"
	)

	BeforeEach(func() {
		port = uint16(8080 + GinkgoParallelProcess())

		certpaths := testhelper.GenerateCerts("metric-server-ca")
		caPath := certpaths.CA()
		certPath := certpaths.Cert("metric-server")
		keyPath := certpaths.Key("metric-server")

		tlsCfg, err := tlsconfig.Build(
			tlsconfig.WithInternalServiceDefaults(),
			tlsconfig.WithIdentityFromFile(certPath, keyPath),
		).Server(
			tlsconfig.WithClientAuthenticationFromFile(caPath),
		)
		Expect(err).NotTo(HaveOccurred())

		reg := prometheus.NewRegistry()
		err = reg.Register(
			prometheus.NewGauge(
				prometheus.GaugeOpts{
					Name: gaugeName,
					Help: "test gauge",
				},
			),
		)
		Expect(err).NotTo(HaveOccurred())

		s = metricserver.New(port, reg, tlsCfg)
		go func() {
			err := s.Run()
			if err != nil {
				GinkgoWriter.Println(err)
			}
		}()

		client = plumbing.NewTLSHTTPClient(
			certPath,
			keyPath,
			caPath,
			"metric-server",
		)
	})

	AfterEach(func() {
		err := s.Stop()
		Expect(err).NotTo(HaveOccurred())
	})

	Context("Addr", func() {
		It("is localhost with given port", func() {
			Expect(s.Addr()).To(Equal(fmt.Sprintf("127.0.0.1:%d", port)))
		})
	})

	Context("Run", func() {
		It("listens via TLS on the Addr", func() {
			resp, err := client.Get("https://" + s.Addr() + "/metrics")
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
		})

		It("serves prometheus metrics from the registry on the Addr", func() {
			Eventually(func(g Gomega) {
				resp, err := client.Get("https://" + s.Addr() + "/metrics")
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(resp.StatusCode).To(Equal(http.StatusOK))
				b, err := io.ReadAll(resp.Body)
				g.Expect(err).NotTo(HaveOccurred())
				body := string(b)
				g.Expect(body).To(ContainSubstring(gaugeName))
			}, 20*time.Second, 1*time.Second)
		})
	})
})
