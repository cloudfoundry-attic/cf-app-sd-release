package main_test

import (
	"io"
	"net/http"
	"os/exec"

	"io/ioutil"

	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Main", func() {

	var (
		session            *gexec.Session
		tempConfigFile     *os.File
		configFileContents string
	)

	BeforeEach(func() {
		configFileContents = `{
			"address": "127.0.0.1",
			"port": "8053"
		}`
	})

	JustBeforeEach(func() {
		var err error
		tempConfigFile, err = ioutil.TempFile(os.TempDir(), "sd")
		Expect(err).ToNot(HaveOccurred())
		_, err = tempConfigFile.Write([]byte(configFileContents))
		Expect(err).ToNot(HaveOccurred())

		startCmd := exec.Command(pathToServer, "-c", tempConfigFile.Name())
		session, err = gexec.Start(startCmd, GinkgoWriter, GinkgoWriter)
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		session.Kill()
		os.Remove(tempConfigFile.Name())
	})

	It("should return a http 200 status", func() {
		Eventually(session).Should(gbytes.Say("Server Started"))

		var reader io.Reader
		request, err := http.NewRequest("GET", "http://127.0.0.1:8053?type=1&name=app-id.internal.local.", reader)
		Expect(err).To(Succeed())

		resp, err := http.DefaultClient.Do(request)
		Expect(err).To(Succeed())

		Expect(err).To(Succeed())
		Expect(resp.StatusCode).To(Equal(http.StatusOK))

		all, err := ioutil.ReadAll(resp.Body)
		Expect(err).To(Succeed())
		Expect(string(all)).To(MatchJSON(`{
					"Status": 0,
					"TC": false,
					"RD": false,
					"RA": false,
					"AD": false,
					"CD": false,
					"Question":
					[
						{
							"name": "app-id.internal.local.",
							"type": 1
						}
					],
					"Answer":
					[
						{
							"name": "app-id.internal.local.",
							"type": 1,
							"TTL":  0,
							"data": "192.168.0.1"
						}
					],
					"Additional": [ ],
					"edns_client_subnet": "0.0.0.0/0"
				}
		`))
	})

	It("accepts interrupt signals and shuts down", func() {
		Eventually(session).Should(gbytes.Say("Server Started"))
		session.Signal(os.Interrupt)

		Eventually(session).Should(gexec.Exit())
		Eventually(session).Should(gbytes.Say("Shutting bosh-dns-adapter down"))
	})

	Context("when a process is already listening on the port", func() {
		var session2 *gexec.Session
		JustBeforeEach(func() {
			startCmd := exec.Command(pathToServer, "-c", tempConfigFile.Name())
			var err error
			session2, err = gexec.Start(startCmd, GinkgoWriter, GinkgoWriter)
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			session2.Kill().Wait()
		})

		It("fails to start", func() {
			Eventually(session2).Should(gexec.Exit(1))
			Eventually(session2.Err).Should(gbytes.Say("Address \\(127.0.0.1:8053\\) not available"))
		})
	})

	Context("when 'type' url param is not provided", func() {
		It("should return a http 400 status", func() {
			Eventually(session).Should(gbytes.Say("Server Started"))

			var reader io.Reader
			request, err := http.NewRequest("GET", "http://127.0.0.1:8053?name=app-id.internal.local.", reader)
			Expect(err).To(Succeed())

			resp, err := http.DefaultClient.Do(request)
			Expect(err).To(Succeed())

			Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))

			all, err := ioutil.ReadAll(resp.Body)
			Expect(err).To(Succeed())
			Expect(string(all)).To(MatchJSON(`{
					"Status": 2,
					"TC": false,
					"RD": false,
					"RA": false,
					"AD": false,
					"CD": false,
					"Question":
					[
						{
							"name": "app-id.internal.local.",
							"type": 0
						}
					],
					"Answer":
					[
						{
							"name": "app-id.internal.local.",
							"type": 1,
							"TTL":  0,
							"data": "192.168.0.1"
						}
					],
					"Additional": [ ],
					"edns_client_subnet": "0.0.0.0/0"
				}`))
		})
	})

	Context("when 'name' url param is not provided", func() {
		It("returns a http 400 status", func() {
			Eventually(session).Should(gbytes.Say("Server Started"))
			var reader io.Reader
			request, err := http.NewRequest("GET", "http://127.0.0.1:8053?type=1", reader)
			Expect(err).To(Succeed())

			resp, err := http.DefaultClient.Do(request)
			Expect(err).To(Succeed())

			Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))

			all, err := ioutil.ReadAll(resp.Body)
			Expect(err).To(Succeed())

			Expect(string(all)).To(MatchJSON(`{
					"Status": 2,
					"TC": false,
					"RD": false,
					"RA": false,
					"AD": false,
					"CD": false,
					"Question":
					[
						{
							"name": "",
							"type": 1
						}
					],
					"Answer":
					[
						{
							"name": "app-id.internal.local.",
							"type": 1,
							"TTL":  0,
							"data": "192.168.0.1"
						}
					],
					"Additional": [ ],
					"edns_client_subnet": "0.0.0.0/0"
				}`))
		})
	})

	Context("when both 'type' and 'name' url params are not provided", func() {
		It("returns a http 400 status", func() {
			Eventually(session).Should(gbytes.Say("Server Started"))
			var reader io.Reader
			request, err := http.NewRequest("GET", "http://127.0.0.1:8053", reader)
			Expect(err).To(Succeed())
			resp, err := http.DefaultClient.Do(request)
			Expect(err).To(Succeed())

			Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))

			all, err := ioutil.ReadAll(resp.Body)
			Expect(err).To(Succeed())

			Expect(string(all)).To(MatchJSON(`{
					"Status": 2,
					"TC": false,
					"RD": false,
					"RA": false,
					"AD": false,
					"CD": false,
					"Question":
					[
						{
							"name": "",
							"type": 0
						}
					],
					"Answer":
					[
						{
							"name": "app-id.internal.local.",
							"type": 1,
							"TTL":  0,
							"data": "192.168.0.1"
						}
					],
					"Additional": [ ],
					"edns_client_subnet": "0.0.0.0/0"
				}`))
		})
	})

	Context("when configured with an invalid port", func() {
		BeforeEach(func() {
			configFileContents = `{
			"address": "127.0.0.1",
			"port": "-1"
		}`
		})

		It("should fail to startup", func() {
			Eventually(session).Should(gexec.Exit(1))
		})
	})

	Context("when configured with an invalid config file path", func() {
		var session2 *gexec.Session
		JustBeforeEach(func() {
			startCmd := exec.Command(pathToServer, "-c", "/non-existent-path")
			var err error
			session2, err = gexec.Start(startCmd, GinkgoWriter, GinkgoWriter)
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			session2.Kill().Wait()
		})

		It("should fail to startup", func() {
			Eventually(session2).Should(gexec.Exit(2))
			Eventually(session2).Should(gbytes.Say("Could not read config file at path '/non-existent-path'"))
		})
	})

	Context("when configured garbage config file content", func() {
		BeforeEach(func() {
			configFileContents = "garbage"
		})

		It("should fail to startup", func() {
			Eventually(session).Should(gexec.Exit(2))
			Eventually(session).Should(gbytes.Say("Could not parse config file at path '%s'", tempConfigFile.Name()))
		})
	})

	Context("when no config file is passed", func() {
		var session2 *gexec.Session
		JustBeforeEach(func() {
			startCmd := exec.Command(pathToServer)
			var err error
			session2, err = gexec.Start(startCmd, GinkgoWriter, GinkgoWriter)
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			session2.Kill().Wait()
		})


		It("should fail to startup", func() {
			Eventually(session2).Should(gexec.Exit(2))
		})
	})
})
