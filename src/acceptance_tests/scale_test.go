package acceptance_tests_test

import (
	"strings"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Scale Acceptance", func() {
	var (
		prefix       string
		orgName      string
		appName      string
		hostName     string
		numInstances int
	)

	BeforeEach(func() {
		prefix = "scale-sd-apps-"

		orgName = prefix + "org"
		spaceName := prefix + "space"
		appName = prefix + "proxy"

		createAndTargetOrgAndSpace(orgName, spaceName)

		By("pushing the app and checking it resolves")
		pushApp(appName, 1)

		proxyGuid := getAppGUID(appName)
		hostName = "http://" + appName + "." + config.AppsDomain + "/dig/" + strings.TrimSpace(proxyGuid) + ".sd-local."
		proxyIPs := digForNumberOfIPs(hostName, 1)

		Expect(proxyIPs).To(ContainElement(getInternalIP(appName, 0)))

		numInstances = 3
	})

	AfterEach(func() {
		Expect(cf.Cf("delete-org", orgName, "-f").Wait(Timeout_Push)).To(gexec.Exit(0))
	})

	Describe("when performing a dns lookup for a domain configured to point to the bosh adapter", func() {
		It("returns the correct results when scaling up and down", func() {
			By("scaling up")
			scaleApp(appName, numInstances)

			By("checking that all three instances are resolvable")
			proxyIPs := digForNumberOfIPs(hostName, numInstances)
			for i := 0; i < numInstances; i++ {
				Expect(proxyIPs).To(ContainElement(getInternalIP(appName, i)))
			}

			By("scaling down")
			scaleApp(appName, 1)

			By("checking that only 0th instance is resolvable")
			proxyIPs = digForNumberOfIPs(hostName, 1)
			Expect(proxyIPs).To(ContainElement(getInternalIP(appName, 0)))
		})
	})
})