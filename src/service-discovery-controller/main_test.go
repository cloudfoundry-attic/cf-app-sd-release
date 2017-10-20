package main_test

import (
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"

	"fmt"
	"time"

	"github.com/nats-io/gnatsd/server"
	"github.com/nats-io/nats"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Main", func() {
	var (
		session      *gexec.Session
		configPath   string
		natsServer   *server.Server
		routeEmitter *nats.Conn
	)

	BeforeEach(func() {
		natsServer = RunNatsServerOnPort(8080)
		configPath = writeConfigFile(`{
			"address":"127.0.0.1",
			"port":"8055",
			"nats":[
				{
					"host":"localhost",
					"port":8080,
					"user":"",
					"pass":""
				}
			]
		}`)
	})

	JustBeforeEach(func() {
		startCmd := exec.Command(pathToServer, "-c", configPath)
		var err error
		session, err = gexec.Start(startCmd, GinkgoWriter, GinkgoWriter)
		Expect(err).ToNot(HaveOccurred())

		Eventually(session).Should(gbytes.Say("Server Started"))

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
			req, err := http.NewRequest("GET", "http://localhost:8055/v1/registration/app-id.internal.local.", nil)
			Expect(err).ToNot(HaveOccurred())
			resp, err := http.DefaultClient.Do(req)
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
	})

	It("should return a http app json", func() {
		req, err := http.NewRequest("GET", "http://localhost:8055/v1/registration/app-id.internal.local.", nil)
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
		req, err := http.NewRequest("GET", "http://localhost:8055/v1/registration/large-id.internal.local.", nil)
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

	Context("when one of the nats urls is invalid", func() {
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
					},
					{
						"host":"localhost",
						"port":8080,
						"user":"",
						"pass":""
					}
				]
			}`)
		})

		It("connects to NATs successfully", func() {
			Eventually(func() string {
				req, err := http.NewRequest("GET", "http://localhost:8055/v1/registration/app-id.internal.local.", nil)
				Expect(err).ToNot(HaveOccurred())
				resp, err := http.DefaultClient.Do(req)
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
