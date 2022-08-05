package debugserver_test

import (
	"fmt"
	"net/http"

	"code.cloudfoundry.org/system-metrics/pkg/debugserver"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Server", Ordered, func() {
	var (
		port uint16 = 8080
		s    *debugserver.DebugServer
	)

	BeforeAll(func() {
		s = debugserver.New(port)
		go func() {
			err := s.Start()
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
		It("is localhost with given port", func() {
			Expect(s.Addr()).To(Equal(fmt.Sprintf("127.0.0.1:%d", port)))
		})
	})

	Context("Start", func() {
		It("listens on pprof endpoints", func() {
			resp, err := http.Get("http://" + s.Addr() + "/debug/pprof/heap")
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
		})
	})
})
