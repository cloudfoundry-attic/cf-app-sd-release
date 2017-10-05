package acceptance_tests_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"fmt"
	"os/exec"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Acceptance", func() {
	Describe("when performing a dns lookup for a domain configured to point to the bosh adapter", func() {
		It("returns the result from the adapter", func() {
			cmd := exec.Command("dig", "app-id.internal.local.", fmt.Sprintf("@%s", allDeployedInstances[0].IP))
			session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
			Expect(err).ToNot(HaveOccurred())

			Eventually(session).Should(gexec.Exit())
			output := string(session.Out.Contents())
			Expect(output).To(ContainSubstring("flags: qr aa rd ra; QUERY: 1, ANSWER: 2, AUTHORITY: 0, ADDITIONAL: 0"))
			Expect(output).To(MatchRegexp("app-id.internal.local.\\s+0\\s+IN\\s+A\\s+192\\.168\\.0\\.1"))
			Expect(output).To(MatchRegexp("app-id.internal.local.\\s+0\\s+IN\\s+A\\s+192\\.168\\.0\\.2"))
		})
	})
})
