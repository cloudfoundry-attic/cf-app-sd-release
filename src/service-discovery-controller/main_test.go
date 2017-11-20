package main_test

import (
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"

	"fmt"
	"time"

	"bosh-dns-adapter/testhelpers"
	"crypto/tls"
	"crypto/x509"

	"github.com/nats-io/gnatsd/server"
	"github.com/nats-io/nats"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Main", func() {
	var (
		session                   *gexec.Session
		configPath                string
		natsServer                *server.Server
		routeEmitter              *nats.Conn
		clientCert                tls.Certificate
		caFile                    string
		serverCert                string
		serverKey                 string
		stalenessThresholdSeconds int
		pruningIntervalSeconds    int
	)

	BeforeEach(func() {
		caFile, serverCert, serverKey, clientCert = testhelpers.GenerateCaAndMutualTlsCerts()
		stalenessThresholdSeconds = 1
		pruningIntervalSeconds = 1

		natsServer = RunNatsServerOnPort(8080)
		configPath = writeConfigFile(fmt.Sprintf(`{
			"address":"127.0.0.1",
			"port":"8055",
			"ca_cert": "%s",
			"server_cert": "%s",
			"server_key": "%s",
			"nats":[
				{
					"host":"localhost",
					"port":8080,
					"user":"",
					"pass":""
				}
			],
			"staleness_threshold_seconds": %d,
			"pruning_interval_seconds": %d
		}`, caFile, serverCert, serverKey, stalenessThresholdSeconds, pruningIntervalSeconds))
	})

	AfterEach(func() {
		session.Kill()
		os.Remove(configPath)
		natsServer.Shutdown()
	})

	Context("when it starts successfully", func() {
		JustBeforeEach(func() {
			startCmd := exec.Command(pathToServer, "-c", configPath)
			var err error
			session, err = gexec.Start(startCmd, GinkgoWriter, GinkgoWriter)
			Expect(err).ToNot(HaveOccurred())

			Eventually(session, 5*time.Second).Should(gbytes.Say("Server Started"))

			routeEmitter = newFakeRouteEmitter("nats://" + natsServer.Addr().String())
			register(routeEmitter, "192.168.0.1", "app-id.internal.local.")
			register(routeEmitter, "192.168.0.2", "app-id.internal.local.")
			register(routeEmitter, "192.168.0.1", "large-id.internal.local.")
			register(routeEmitter, "192.168.0.2", "large-id.internal.local.")
			register(routeEmitter, "192.168.0.3", "large-id.internal.local.")
			register(routeEmitter, "192.168.0.4", "large-id.internal.local.")
			register(routeEmitter, "192.168.0.5", "large-id.internal.local.")
			register(routeEmitter, "192.168.0.6", "large-id.internal.local.")
			register(routeEmitter, "192.168.0.7", "large-id.internal.local.")
			register(routeEmitter, "192.168.0.8", "large-id.internal.local.")
			register(routeEmitter, "192.168.0.9", "large-id.internal.local.")
			register(routeEmitter, "192.168.0.10", "large-id.internal.local.")
			register(routeEmitter, "192.168.0.11", "large-id.internal.local.")
			register(routeEmitter, "192.168.0.12", "large-id.internal.local.")
			register(routeEmitter, "192.168.0.13", "large-id.internal.local.")
			Expect(routeEmitter.Flush()).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			session.Kill()
			os.Remove(configPath)
			routeEmitter.Close()
			natsServer.Shutdown()
		})

		It("accepts interrupt signals and shuts down", func() {
			session.Signal(os.Interrupt)

			Eventually(session).Should(gexec.Exit())
			Eventually(session).Should(gbytes.Say("Shutting service-discovery-controller down"))
		})

		It("should not return ips for unregistered domains", func() {
			unregister(routeEmitter, "192.168.0.1", "app-id.internal.local.")
			Expect(routeEmitter.Flush()).ToNot(HaveOccurred())

			Eventually(func() string {
				resp, err := NewClient(testhelpers.CertPool(caFile), clientCert).Get("https://localhost:8055/v1/registration/app-id.internal.local.")
				Expect(err).ToNot(HaveOccurred())
				respBody, err := ioutil.ReadAll(resp.Body)
				Expect(err).ToNot(HaveOccurred())
				return string(respBody)
			}).Should(MatchJSON(`{
				"env": "",
				"hosts": [
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

			sessionContents := string(session.Out.Contents())
			Expect(sessionContents).To(MatchRegexp(`HTTPServer access.*\"ip_address\\":\\"192.168.0.2\\".*\"serviceKey\":\"app-id.internal.local.\"`))
		})

		It("should return a http app json", func() {
			resp, err := NewClient(testhelpers.CertPool(caFile), clientCert).Get("https://localhost:8055/v1/registration/app-id.internal.local.")
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
			resp, err := NewClient(testhelpers.CertPool(caFile), clientCert).Get("https://localhost:8055/v1/registration/large-id.internal.local.")
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

		It("eventually removes stale routes", func() {
			client := NewClient(testhelpers.CertPool(caFile), clientCert)
			waitDuration := time.Duration(stalenessThresholdSeconds+pruningIntervalSeconds+1) * time.Second

			Eventually(func() []byte {
				resp, err := client.Get("https://localhost:8055/v1/registration/app-id.internal.local.")
				Expect(err).ToNot(HaveOccurred())

				respBody, err := ioutil.ReadAll(resp.Body)
				Expect(err).ToNot(HaveOccurred())

				return respBody
			}, waitDuration).Should(MatchJSON(`{ "env": "", "hosts": [], "service": "" }`))
		})

		Context("when we hit the /routes endpoint", func() {
			It("should return a map of all hostnames to ips", func() {
				resp, err := NewClient(testhelpers.CertPool(caFile), clientCert).Get("https://localhost:8055/routes")
				Expect(err).ToNot(HaveOccurred())
				respBody, err := ioutil.ReadAll(resp.Body)
				Expect(err).ToNot(HaveOccurred())

				Expect(respBody).To(Or(MatchJSON(`{
					"addresses": [{
						"hostname": "app-id.internal.local.",
						"ips": [
							"192.168.0.1",
							"192.168.0.2"
							]
					}, {
						"hostname": "large-id.internal.local.",
						"ips": [
							"192.168.0.1",
							"192.168.0.2",
							"192.168.0.3",
							"192.168.0.4",
							"192.168.0.5",
							"192.168.0.6",
							"192.168.0.7",
							"192.168.0.8",
							"192.168.0.9",
							"192.168.0.10",
							"192.168.0.11",
							"192.168.0.12",
							"192.168.0.13"
							]
						}
					]
				}`),
					MatchJSON(`{
					"addresses": [
						{
							"hostname": "large-id.internal.local.",
							"ips": [
								"192.168.0.1",
								"192.168.0.2",
								"192.168.0.3",
								"192.168.0.4",
								"192.168.0.5",
								"192.168.0.6",
								"192.168.0.7",
								"192.168.0.8",
								"192.168.0.9",
								"192.168.0.10",
								"192.168.0.11",
								"192.168.0.12",
								"192.168.0.13"
							]
						}, {
							"hostname": "app-id.internal.local.",
							"ips": [
								"192.168.0.1",
								"192.168.0.2"
							]
						}
					]
				}`),
				))
			})
		})

		Context("when one of the nats urls is invalid", func() {
			BeforeEach(func() {
				os.Remove(configPath)
				configPath = writeConfigFile(fmt.Sprintf(`{
					"address":"127.0.0.1",
					"port":"8055",
					"ca_cert": "%s",
					"server_cert": "%s",
					"server_key": "%s",
					"nats":[
						{
							"host":"garbage",
							"port":8081,
							"user":"who",
							"pass":"what"
						},
						{
							"host":"localhost",
							"port":8080,
							"user":"",
							"pass":""
						}
					],
					"staleness_threshold_seconds": %d,
					"pruning_interval_seconds": %d
				}`, caFile, serverCert, serverKey, stalenessThresholdSeconds, pruningIntervalSeconds))
			})

			It("connects to NATs successfully", func() {
				Eventually(func() string {
					resp, err := NewClient(testhelpers.CertPool(caFile), clientCert).Get("https://localhost:8055/v1/registration/app-id.internal.local.")
					Expect(err).ToNot(HaveOccurred())
					respBody, err := ioutil.ReadAll(resp.Body)
					Expect(err).ToNot(HaveOccurred())
					return string(respBody)
				}).Should(MatchJSON(`{
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
		})

		Context("when disconnected from the nats server", func() {
			var (
				client       *http.Client
				waitDuration time.Duration
			)

			BeforeEach(func() {
				client = NewClient(testhelpers.CertPool(caFile), clientCert)
				waitDuration = time.Duration(stalenessThresholdSeconds+pruningIntervalSeconds+1) * time.Second
			})
			It("does not prune stale entries", func() {
				By("stopping the nats server and still returning routes past staleness threshold", func() {
					natsServer.Shutdown()
					Consistently(func() []byte {
						resp, err := client.Get("https://localhost:8055/v1/registration/app-id.internal.local.")
						Expect(err).ToNot(HaveOccurred())

						respBody, err := ioutil.ReadAll(resp.Body)
						Expect(err).ToNot(HaveOccurred())

						return respBody
					}, waitDuration).Should(MatchJSON(`{
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

				By("resuming pruning when nats server is back up", func() {
					natsServer = RunNatsServerOnPort(8080)
					Eventually(func() []byte {
						resp, err := client.Get("https://localhost:8055/v1/registration/app-id.internal.local.")
						Expect(err).ToNot(HaveOccurred())

						respBody, err := ioutil.ReadAll(resp.Body)
						Expect(err).ToNot(HaveOccurred())

						return respBody
					}, waitDuration).Should(MatchJSON(`{
					"env": "",
					"hosts": [],
					"service": ""
				}`))
				})

			})
		})
	})

	Context("when none of the nats urls are valid", func() {
		BeforeEach(func() {
			os.Remove(configPath)
			configPath = writeConfigFile(`{
				"address":"127.0.0.1",
				"port":"8055",
				"nats":[
					{
						"host":"garbage",
						"port":8081,
						"user":"who",
						"pass":"what"
					}
				]
			}`)
		})

		It("fails to start successfully", func() {
			startCmd := exec.Command(pathToServer, "-c", configPath)
			var err error
			session, err = gexec.Start(startCmd, GinkgoWriter, GinkgoWriter)
			Expect(err).ToNot(HaveOccurred())
			Eventually(session, 5*time.Second).Should(gexec.Exit(2))
		})
	})
})

func register(routeEmitter *nats.Conn, ip string, url string) {
	natsRegistryMsg := nats.Msg{
		Subject: "service-discovery.register",
		Data:    []byte(fmt.Sprintf(`{"host": "%s","uris":["%s"]}`, ip, url)),
	}

	Expect(routeEmitter.PublishMsg(&natsRegistryMsg)).ToNot(HaveOccurred())
}

func unregister(routeEmitter *nats.Conn, ip string, url string) {
	natsRegistryMsg := nats.Msg{
		Subject: "service-discovery.unregister",
		Data:    []byte(fmt.Sprintf(`{"host": "%s","uris":["%s"]}`, ip, url)),
	}

	Expect(routeEmitter.PublishMsg(&natsRegistryMsg)).ToNot(HaveOccurred())
}

func newFakeRouteEmitter(natsUrl string) *nats.Conn {
	natsClient, err := nats.Connect(natsUrl, nats.ReconnectWait(1*time.Nanosecond))
	Expect(err).NotTo(HaveOccurred())
	return natsClient
}

func writeConfigFile(configJson string) string {
	configFile, err := ioutil.TempFile(os.TempDir(), "sd_config")
	Expect(err).ToNot(HaveOccurred())

	configPath := configFile.Name()

	err = ioutil.WriteFile(configPath, []byte(configJson), os.ModePerm)
	Expect(err).ToNot(HaveOccurred())

	return configPath
}

func NewClient(caCertPool *x509.CertPool, cert tls.Certificate) *http.Client {
	tlsConfig := &tls.Config{
		MinVersion:   tls.VersionTLS12,
		ClientCAs:    caCertPool,
		RootCAs:      caCertPool,
		Certificates: []tls.Certificate{cert},
	}

	tlsConfig.BuildNameToCertificate()
	tlsConfig.ServerName = "service-discovery-controller.internal"

	tr := &http.Transport{
		TLSClientConfig: tlsConfig,
	}

	return &http.Client{Transport: tr}
}
