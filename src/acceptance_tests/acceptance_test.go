package acceptance_tests_test

import (
	"path/filepath"
	"time"

	"bytes"
	"encoding/json"
	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"io/ioutil"
	"net/http"
	"strings"
)

const Timeout_Push = 2 * time.Minute

var (
	appsDir string
	prefix  string
	orgName string
)
var _ = Describe("Acceptance", func() {

	BeforeEach(func() {
		prefix = "sd-apps"

		orgName = prefix + "org" // cf-pusher expects this name
		Expect(cf.Cf("create-org", orgName).Wait(Timeout_Push)).To(gexec.Exit(0))
		Expect(cf.Cf("target", "-o", orgName).Wait(Timeout_Push)).To(gexec.Exit(0))

		spaceName := prefix + "space" // cf-pusher expects this name
		Expect(cf.Cf("create-space", spaceName, "-o", orgName).Wait(Timeout_Push)).To(gexec.Exit(0))
		Expect(cf.Cf("target", "-o", orgName, "-s", spaceName).Wait(Timeout_Push)).To(gexec.Exit(0))
	})

	AfterEach(func() {
		Expect(cf.Cf("delete-org", orgName, "-f").Wait(Timeout_Push)).To(gexec.Exit(0))
	})

	Describe("when performing a dns lookup for a domain configured to point to the bosh adapter", func() {
		It("returns the result from the adapter", func() {
			pushProxy("proxy")

			session := cf.Cf("app", "proxy", "--guid").Wait(10 * time.Second)

			proxyGuid := string(session.Out.Contents())

			resp, err := http.Get("http://proxy." + config.AppsDomain + "/dig/" + strings.TrimSpace(proxyGuid) + ".sd-local.")

			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			ipsJson, err := ioutil.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())

			proxyIps := &[]string{}

			println(string(ipsJson))
			err = json.Unmarshal(bytes.TrimSpace(ipsJson), proxyIps)
			Expect(err).NotTo(HaveOccurred())

			session = cf.Cf("ssh", "proxy", "-c", "echo $CF_INSTANCE_INTERNAL_IP").Wait(10 * time.Second)
			proxyContainerIp := string(session.Out.Contents())

			Expect(*proxyIps).To(ContainElement(strings.TrimSpace(proxyContainerIp)))
		})
	})
})

func appDir(appType string) string {
	return filepath.Join(appsDir, appType)
}

func pushProxy(appName string) {
	Expect(cf.Cf(
		"push", appName,
		"-p", appDir("proxy"),
		"-f", defaultManifest("proxy"),
	).Wait(Timeout_Push)).To(gexec.Exit(0))
}

func defaultManifest(appType string) string {
	return filepath.Join(appDir(appType), "manifest.yml")
}
