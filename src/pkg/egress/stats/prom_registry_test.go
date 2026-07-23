package stats_test

import (
	"code.cloudfoundry.org/system-metrics-release/src/pkg/egress/stats"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus"
)

var _ = Describe("Prometheus Registry", func() {
	var (
		registry   *stats.PromRegistry
		registerer *stubRegisterer
	)

	BeforeEach(func() {
		registerer = newStubRegisterer()
		registry = stats.NewPromRegistry(registerer)
	})

	It("registers the gauge when it's first gotten", func() {
		gauge := toPromGauge(registry.Get("metric_name", "origin", "unit", nil))
		expectGauge(gauge, "metric_name", map[string]string{"unit": "unit"})

		var registered prometheus.Gauge
		Eventually(registerer.gauges).Should(Receive(&registered))

		expectGauge(registered, "metric_name", map[string]string{"unit": "unit"})
	})

	It("doesn't reregister gauges", func() {
		gauge := toPromGauge(registry.Get("metric_name", "origin", "unit", nil))
		expectGauge(gauge, "metric_name", map[string]string{"unit": "unit"})

		gauge = toPromGauge(registry.Get("metric_name", "origin", "unit", nil))
		expectGauge(gauge, "metric_name", map[string]string{"unit": "unit"})

		Expect(registerer.gauges).To(HaveLen(1))
	})

	It("gauges with different tags are different gauges", func() {
		gauge := toPromGauge(registry.Get("metric_name", "origin", "unit", map[string]string{"foo": "bar2"}))
		expectGauge(gauge, "metric_name", map[string]string{"foo": "bar2", "unit": "unit"})

		gauge = toPromGauge(registry.Get("metric_name", "origin", "unit", map[string]string{"foo": "bar"}))
		expectGauge(gauge, "metric_name", map[string]string{"foo": "bar", "unit": "unit"})

		Expect(registerer.gauges).To(HaveLen(2))
	})
})

// gaugeDesc is a plain, comparable projection of a collected gauge's public
// contract (its exported name and const labels).
type gaugeDesc struct {
	Name   string
	Labels map[string]string
}

// expectGauge collects the gauge through a real registry and compares its
// public contract against the expected name and const labels as a single object.
func expectGauge(g prometheus.Gauge, name string, constLabels map[string]string) {
	GinkgoHelper()

	reg := prometheus.NewRegistry()
	Expect(reg.Register(g)).To(Succeed())

	families, err := reg.Gather()
	Expect(err).NotTo(HaveOccurred())
	Expect(families).To(HaveLen(1))
	Expect(families[0].GetMetric()).To(HaveLen(1))

	labels := map[string]string{}
	for _, p := range families[0].GetMetric()[0].GetLabel() {
		labels[p.GetName()] = p.GetValue()
	}

	actual := gaugeDesc{Name: families[0].GetName(), Labels: labels}
	Expect(actual).To(Equal(gaugeDesc{Name: name, Labels: constLabels}))
}

func toPromGauge(g stats.Gauge) prometheus.Gauge {
	gauge, ok := g.(prometheus.Gauge)
	Expect(ok).To(BeTrue())
	return gauge
}

type stubRegisterer struct {
	gauges chan prometheus.Gauge
}

func newStubRegisterer() *stubRegisterer {
	return &stubRegisterer{
		gauges: make(chan prometheus.Gauge, 100),
	}
}

func (r *stubRegisterer) Register(c prometheus.Collector) error {
	gauge, ok := c.(prometheus.Gauge)
	Expect(ok).To(BeTrue())

	r.gauges <- gauge
	return nil
}

func (r *stubRegisterer) MustRegister(...prometheus.Collector) {

}

func (r *stubRegisterer) Unregister(prometheus.Collector) bool {
	return false
}
