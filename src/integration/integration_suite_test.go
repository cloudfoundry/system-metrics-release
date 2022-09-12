package integration_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var agentPath string

func TestIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Integration Test Suite")
}

var _ = SynchronizedBeforeSuite(func() []byte {
	// runs on first process
	path, err := gexec.Build("../cmd/agentd")
	Expect(err).NotTo(HaveOccurred())
	DeferCleanup(gexec.CleanupBuildArtifacts)

	return []byte(path)
}, func(path []byte) {
	// runs on all processes
	agentPath = string(path)
})
