package main_test

import (
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
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

	It("should return a http app json", func() {
		Eventually(session).Should(gbytes.Say("Server Started"))

		req, err := http.NewRequest("GET", "http://localhost:8054/v1/registration/app-id.internal.local", nil)
		Expect(err).ToNot(HaveOccurred())
		resp, err := http.DefaultClient.Do(req)
		Expect(err).ToNot(HaveOccurred())
		respBody, err := ioutil.ReadAll(resp.Body)
		Expect(err).ToNot(HaveOccurred())

		Expect(respBody).To(MatchJSON(`{
			"env": "",
			"hosts": [
			{
				"ip_address": "192.168.0.1",
				"last_check_in": "",
				"port": 0,
				"revision": "",
				"service": "",
				"service_repo_name": "",
				"tags": {}
			},
			{
				"ip_address": "192.168.0.2",
				"last_check_in": "",
				"port": 0,
				"revision": "",
				"service": "",
				"service_repo_name": "",
				"tags": {}
			}],
			"service": ""
		}`))
	})

	It("should return a http large json", func() {
		Eventually(session).Should(gbytes.Say("Server Started"))

		req, err := http.NewRequest("GET", "http://localhost:8054/v1/registration/large-id.internal.local", nil)
		Expect(err).ToNot(HaveOccurred())
		resp, err := http.DefaultClient.Do(req)
		Expect(err).ToNot(HaveOccurred())
		respBody, err := ioutil.ReadAll(resp.Body)
		Expect(err).ToNot(HaveOccurred())

		Expect(respBody).To(MatchJSON(`{
			"env": "",
			"hosts": [
			{
				"ip_address": "192.168.0.1",
				"last_check_in": "",
				"port": 0,
				"revision": "",
				"service": "",
				"service_repo_name": "",
				"tags": {}
			},
			{
				"ip_address": "192.168.0.2",
				"last_check_in": "",
				"port": 0,
				"revision": "",
				"service": "",
				"service_repo_name": "",
				"tags": {}
			},
			{
				"ip_address": "192.168.0.3",
				"last_check_in": "",
				"port": 0,
				"revision": "",
				"service": "",
				"service_repo_name": "",
				"tags": {}
			},
			{
				"ip_address": "192.168.0.4",
				"last_check_in": "",
				"port": 0,
				"revision": "",
				"service": "",
				"service_repo_name": "",
				"tags": {}
			},
			{
				"ip_address": "192.168.0.5",
				"last_check_in": "",
				"port": 0,
				"revision": "",
				"service": "",
				"service_repo_name": "",
				"tags": {}
			},
			{
				"ip_address": "192.168.0.6",
				"last_check_in": "",
				"port": 0,
				"revision": "",
				"service": "",
				"service_repo_name": "",
				"tags": {}
			},
			{
				"ip_address": "192.168.0.7",
				"last_check_in": "",
				"port": 0,
				"revision": "",
				"service": "",
				"service_repo_name": "",
				"tags": {}
			},
			{
				"ip_address": "192.168.0.8",
				"last_check_in": "",
				"port": 0,
				"revision": "",
				"service": "",
				"service_repo_name": "",
				"tags": {}
			},
			{
				"ip_address": "192.168.0.9",
				"last_check_in": "",
				"port": 0,
				"revision": "",
				"service": "",
				"service_repo_name": "",
				"tags": {}
			},
			{
				"ip_address": "192.168.0.10",
				"last_check_in": "",
				"port": 0,
				"revision": "",
				"service": "",
				"service_repo_name": "",
				"tags": {}
			},
			{
				"ip_address": "192.168.0.11",
				"last_check_in": "",
				"port": 0,
				"revision": "",
				"service": "",
				"service_repo_name": "",
				"tags": {}
			},
			{
				"ip_address": "192.168.0.12",
				"last_check_in": "",
				"port": 0,
				"revision": "",
				"service": "",
				"service_repo_name": "",
				"tags": {}
			},
						{
				"ip_address": "192.168.0.13",
				"last_check_in": "",
				"port": 0,
				"revision": "",
				"service": "",
				"service_repo_name": "",
				"tags": {}
			}
			],
			"service": ""
		}`))
	})
})
