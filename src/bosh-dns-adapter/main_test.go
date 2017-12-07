package main_test

import (
	"bosh-dns-adapter/testhelpers"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	"github.com/onsi/gomega/ghttp"
	"github.com/onsi/gomega/types"
	"code.cloudfoundry.org/cf-networking-helpers/testsupport/metrics"
)

var _ = Describe("Main", func() {
	var (
		session                                *gexec.Session
		tempConfigFile                         *os.File
		configFileContents                     string
		fakeServiceDiscoveryControllerServer   *ghttp.Server
		fakeServiceDiscoveryControllerResponse http.HandlerFunc
		dnsAdapterAddress                      string
		dnsAdapterPort                         string
		fakeMetron                             metrics.FakeMetron
	)

	BeforeEach(func() {
		fakeMetron = metrics.NewFakeMetron()

		fakeServiceDiscoveryControllerResponse = ghttp.CombineHandlers(
			ghttp.VerifyRequest("GET", "/v1/registration/app-id.internal.local."),
			ghttp.RespondWith(200, `{
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
					}],
					"service": ""
				}`),
		)
		dnsAdapterAddress = "127.0.0.1"
		dnsAdapterPort = "8053"

	})

	JustBeforeEach(func() {
		var err error
		caFileName, clientCertFileName, clientKeyFileName, serverCert := testhelpers.GenerateCaAndMutualTlsCerts()

		fakeServiceDiscoveryControllerServer = ghttp.NewUnstartedServer()
		fakeServiceDiscoveryControllerServer.HTTPTestServer.TLS = &tls.Config{}
		fakeServiceDiscoveryControllerServer.HTTPTestServer.TLS.RootCAs = testhelpers.CertPool(caFileName)
		fakeServiceDiscoveryControllerServer.HTTPTestServer.TLS.ClientCAs = testhelpers.CertPool(caFileName)
		fakeServiceDiscoveryControllerServer.HTTPTestServer.TLS.ClientAuth = tls.RequireAndVerifyClientCert
		fakeServiceDiscoveryControllerServer.HTTPTestServer.TLS.CipherSuites = []uint16{tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256}
		fakeServiceDiscoveryControllerServer.HTTPTestServer.TLS.PreferServerCipherSuites = true
		fakeServiceDiscoveryControllerServer.HTTPTestServer.TLS.Certificates = []tls.Certificate{serverCert}

		fakeServiceDiscoveryControllerServer.AppendHandlers(fakeServiceDiscoveryControllerResponse)
		fakeServiceDiscoveryControllerServer.HTTPTestServer.StartTLS()

		urlParts := strings.Split(fakeServiceDiscoveryControllerServer.URL(), ":")

		configFileContents = fmt.Sprintf(`{
			"address": "%s",
			"port": "%s",
			"service_discovery_controller_address": "%s",
			"service_discovery_controller_port": "%s",
			"client_cert": "%s",
			"client_key": "%s",
			"ca_cert": "%s",
			"metron_port": %d,
			"metrics_emit_seconds": 2
		}`, dnsAdapterAddress, dnsAdapterPort, strings.TrimPrefix(urlParts[1], "//"), urlParts[2],
			clientCertFileName,
			clientKeyFileName,
			caFileName,
			fakeMetron.Port())

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

		fakeServiceDiscoveryControllerServer.Close()
	})

	It("should return a http 200 status", func() {

		Eventually(session).Should(gbytes.Say("Server Started"))

		var reader io.Reader
		request, err := http.NewRequest("GET", "http://127.0.0.1:8053?type=1&name=app-id.internal.local.", reader)
		Expect(err).To(Succeed())

		resp, err := http.DefaultClient.Do(request)
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

	Describe("emitting metrics", func() {
		withName := func(name string) types.GomegaMatcher {
			return WithTransform(func(ev metrics.Event) string {
				return ev.Name
			}, Equal(name))
		}
		withOrigin := func(origin string) types.GomegaMatcher {
			return WithTransform(func(ev metrics.Event) string {
				return ev.Origin
			}, Equal(origin))
		}

		It("emits an uptime metric", func() {
			Eventually(fakeMetron.AllEvents, "5s").Should(ContainElement(SatisfyAll(
				withName("uptime"),
				withOrigin("bosh-dns-adapter"),
			)))
		})
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
			Eventually(session2, 5*time.Second).Should(gexec.Exit(1))
			Eventually(session2.Err).Should(gbytes.Say("Address \\(127.0.0.1:8053\\) not available"))
		})
	})

	Context("when 'type' url param is not provided", func() {
		It("should default to type A record", func() {
			Eventually(session).Should(gbytes.Say("Server Started"))

			var reader io.Reader
			request, err := http.NewRequest("GET", "http://127.0.0.1:8053?name=app-id.internal.local.", reader)
			Expect(err).To(Succeed())

			resp, err := http.DefaultClient.Do(request)
			Expect(err).To(Succeed())

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
					"Answer": [ ],
					"Additional": [ ],
					"edns_client_subnet": "0.0.0.0/0"
				}`))
		})
	})

	Context("when configured with an invalid port", func() {
		BeforeEach(func() {
			dnsAdapterPort = "-1"
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
			dnsAdapterAddress = `"garbage`
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

	Context("when requesting anything but an A record", func() {
		It("should return a successful response with no answers", func() {
			Eventually(session).Should(gbytes.Say("Server Started"))
			request, err := http.NewRequest("GET", "http://127.0.0.1:8053?type=16&name=app-id.internal.local.", nil)
			Expect(err).ToNot(HaveOccurred())

			resp, err := http.DefaultClient.Do(request)
			Expect(err).ToNot(HaveOccurred())

			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			all, err := ioutil.ReadAll(resp.Body)
			Expect(err).ToNot(HaveOccurred())

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
							"type": 16
						}
					],
					"Answer": [ ],
					"Additional": [ ],
					"edns_client_subnet": "0.0.0.0/0"
				}`))
		})
	})

	Context("when the service discovery controller returns non-successful", func() {
		BeforeEach(func() {
			fakeServiceDiscoveryControllerResponse = ghttp.CombineHandlers(
				ghttp.VerifyRequest("GET", "/v1/registration/app-id.internal.local."),
				ghttp.RespondWith(404, `{ }`),
			)
		})

		It("returns a 500 and an error", func() {
			Eventually(session).Should(gbytes.Say("Server Started"))
			var reader io.Reader

			request, err := http.NewRequest("GET", "http://127.0.0.1:8053?type=1&name=app-id.internal.local.", reader)
			Expect(err).To(Succeed())

			resp, err := http.DefaultClient.Do(request)
			Expect(err).To(Succeed())

			Expect(resp.StatusCode).To(Equal(http.StatusInternalServerError))
		})

	})
})
