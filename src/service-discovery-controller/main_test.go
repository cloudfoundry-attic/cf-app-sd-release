package main_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"os"
	"github.com/onsi/gomega/gexec"
	"os/exec"
)

var _ = Describe("Main", func() {

	var (
		session *gexec.Session
	)

	JustBeforeEach(func() {
		var err error
		startCmd := exec.Command(pathToServer)
		session, err = gexec.Start(startCmd, GinkgoWriter, GinkgoWriter)
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		session.Kill()
	})

	It("accepts interrupt signals and shuts down", func() {
		Eventually(session).Should(gbytes.Say("Server Started"))
		session.Signal(os.Interrupt)

		Eventually(session).Should(gexec.Exit())
		Eventually(session).Should(gbytes.Say("Shutting service-discovery-controller down"))
	})

})
