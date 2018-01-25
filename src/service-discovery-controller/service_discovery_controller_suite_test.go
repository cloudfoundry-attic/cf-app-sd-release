package main_test

import (
	"code.cloudfoundry.org/cf-networking-helpers/testsupport/metrics"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
	"time"

	"github.com/nats-io/gnatsd/server"
	gnatsd "github.com/nats-io/gnatsd/test"
	"github.com/onsi/gomega/gexec"
	"github.com/onsi/gomega/types"
)

func TestServiceDiscoveryController(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ServiceDiscoveryController Suite")
}

var pathToServer string

var _ = SynchronizedBeforeSuite(func() []byte {
	path, err := gexec.Build("service-discovery-controller")
	Expect(err).NotTo(HaveOccurred())
	SetDefaultEventuallyTimeout(2 * time.Second)
	return []byte(path)
}, func(data []byte) {
	pathToServer = string(data)
})

var _ = SynchronizedAfterSuite(func() {
}, func() {
	gexec.CleanupBuildArtifacts()
})

func RunNatsServerOnPort(port int) *server.Server {
	opts := gnatsd.DefaultTestOptions
	opts.Port = port
	return gnatsd.RunServer(&opts)
}

var HaveName = func(name string) types.GomegaMatcher {
	return WithTransform(func(ev metrics.Event) string {
		return ev.Name
	}, Equal(name))
}
