package app_test

import (
	"context"
	"fmt"
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"code.cloudfoundry.org/system-metrics-release/src/v2/cmd/system-metrics-agent/app"
)

const (
	PORT = 8000
)

var _ = Describe("Server", func() {
	var srv *app.Server

	BeforeEach(func() {
		cfg := &app.Config{
			Port: PORT + uint16(GinkgoParallelProcess()),
		}
		srv = app.NewServer(cfg)
	})

	Describe("Run", func() {
		var cancel context.CancelFunc

		BeforeEach(func() {
			var ctx context.Context
			ctx, cancel = context.WithCancel(context.Background())
			go srv.Run(ctx) //nolint:errcheck
		})

		AfterEach(func() {
			cancel()
		})

		It("listens on the provided port", func() {
			resp, err := http.Get(fmt.Sprintf("http://localhost:%d/metrics", PORT+uint16(GinkgoParallelProcess())))
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(200))
			resp.Body.Close()
		})

		XIt("returns a prometheus response", func() {
		})
	})
})
