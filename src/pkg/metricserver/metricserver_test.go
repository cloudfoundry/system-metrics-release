package metricserver_test

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"

	"code.cloudfoundry.org/system-metrics/internal/testhelper"
	"code.cloudfoundry.org/system-metrics/pkg/metricserver"
	"code.cloudfoundry.org/tlsconfig"

	"github.com/prometheus/client_golang/prometheus"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("MetricServer", Ordered, func() {
	var (
		port      uint16 = 8080
		s         *metricserver.MetricServer
		certpaths *testhelper.TestCerts
		reg       *prometheus.Registry
	)

	BeforeAll(func() {
		certpaths = testhelper.GenerateCerts("metric-server-ca")
		reg = prometheus.NewRegistry()
		s = metricserver.New(port, tlsCfg(certpaths.CA(), certpaths.Cert("metric-server"), certpaths.Key("metric-server")), reg)
		go func() {
			err := s.Run()
			if err != nil {
				fmt.Fprintln(GinkgoWriter, err)
			}
		}()
	})

	AfterAll(func() {
		err := s.Stop()
		Expect(err).NotTo(HaveOccurred())
	})

	Context("Addr", func() {
		It("listens on given port", func() {
			Expect(s.Addr()).To(Equal(fmt.Sprintf(":%d", port)))
		})
	})

	Context("Start", func() {
		var client *http.Client
		BeforeAll(func() {
			client = &http.Client{
				Transport: &http.Transport{
					TLSClientConfig: &tls.Config{
						Certificates:       []tls.Certificate{cert},
						InsecureSkipVerify: true,
					},
				},
			}
		})

		It("listens on the metrics endpoint", func() {
			resp, err := client.Get("https://" + s.Addr() + "/metrics")
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
		})

		It("prometheus registry handles the responses", func() {
			g := prometheus.NewGauge(prometheus.GaugeOpts{
				Name: "test",
				Help: "Test value.",
			})
			reg.MustRegister(g)
			g.Set(10)

			resp, err := client.Get("https://" + s.Addr() + "/metrics")
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			body, err := ioutil.ReadAll(resp.Body)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(body) > 0).To(BeTrue())
			bodyStr := string(body)
			Expect(bodyStr).To(ContainSubstring("# HELP test Test value."))
			Expect(bodyStr).To(ContainSubstring("# TYPE test gauge"))
			Expect(bodyStr).To(ContainSubstring("test 10"))
		})
	})
})

func tlsCfg(capath, certpath, keypath string) *tls.Config {
	tlsCfg, err := tlsconfig.Build(
		tlsconfig.WithInternalServiceDefaults(),
		tlsconfig.WithIdentityFromFile(certpath, keypath),
	).Server(
		tlsconfig.WithClientAuthenticationFromFile(capath),
	)
	Expect(err).NotTo(HaveOccurred())
	return tlsCfg
}
