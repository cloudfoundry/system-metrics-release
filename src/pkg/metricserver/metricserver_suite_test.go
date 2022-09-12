package metricserver_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestMetricServer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Metric Server Suite")
}
