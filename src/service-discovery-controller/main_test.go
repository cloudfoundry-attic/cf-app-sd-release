package main_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"service-discovery-controller/config"
	"test"
	"test/common"

	"github.com/nats-io/nats"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

func routeExists(routesEndpoint, routeName string) (bool, error) {
	resp, err := http.Get(routesEndpoint)
	if err != nil {
		fmt.Println("Failed to get from routes endpoint")
		return false, err
	}
	switch resp.StatusCode {
	case http.StatusOK:
		bytes, err := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		Expect(err).ToNot(HaveOccurred())
		routes := make(map[string]interface{})
		err = json.Unmarshal(bytes, &routes)
		Expect(err).ToNot(HaveOccurred())

		_, found := routes[routeName]
		return found, nil

	default:
		return false, errors.New("Didn't get an OK response")
	}
}

func appRegistered(routesUri string, app *common.TestApp) bool {
	routeFound, err := routeExists(routesUri, string(app.Urls()[0]))
	return err == nil && routeFound
}

func appUnregistered(routesUri string, app *common.TestApp) bool {
	routeFound, err := routeExists(routesUri, string(app.Urls()[0]))
	return err == nil && !routeFound
}

func newMessageBus(c *config.Config) (*nats.Conn, error) {
	natsMembers := make([]string, len(c.Nats))
	options := nats.DefaultOptions
	for _, info := range c.Nats {
		uri := url.URL{
			Scheme: "nats",
			User:   url.UserPassword(info.User, info.Pass),
			Host:   fmt.Sprintf("%s:%d", info.Host, info.Port),
		}
		natsMembers = append(natsMembers, uri.String())
	}
	options.Servers = natsMembers
	return options.Connect()
}

var _ = Describe("Main", func() {

	var (
		session      *gexec.Session
		pathToConfig string
		mainConfig   *config.Config
		configJson   []byte

		natsPort   uint16
		natsRunner *test.NATSRunner

		mbusClient *nats.Conn
	)

	BeforeEach(func() {
		configFile, err := ioutil.TempFile(os.TempDir(), "sd_config")
		Expect(err).ToNot(HaveOccurred())
		pathToConfig = configFile.Name()
		mainConfig = &config.Config{}
		mainConfig.Address = "127.0.0.1"
		mainConfig.Port = "8055"
		mainConfig.LogLevel = "debug"

		natsPort = uint16(62000 + GinkgoParallelNode())
		natsRunner = test.NewNATSRunner(int(natsPort))
		natsRunner.Start()

		mainConfig.Nats = []config.NatsConfig{
			{
				Host: "localhost",
				Port: natsPort,
				User: "nats",
				Pass: "nats",
			},
		}
	})

	JustBeforeEach(func() {
		var err error

		configJson, err = json.Marshal(mainConfig)
		Expect(err).NotTo(HaveOccurred())

		err = ioutil.WriteFile(pathToConfig, configJson, os.ModePerm)
		Expect(err).ToNot(HaveOccurred())

		startCmd := exec.Command(pathToServer, "-c", pathToConfig)
		session, err = gexec.Start(startCmd, GinkgoWriter, GinkgoWriter)
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		if natsRunner != nil {
			natsRunner.Stop()
		}
		session.Kill()
		os.Remove(pathToConfig)
	})

	It("accepts interrupt signals and shuts down", func() {
		Eventually(session).Should(gbytes.Say("Server Started"))
		session.Signal(os.Interrupt)

		Eventually(session).Should(gexec.Exit())
		Eventually(session).Should(gbytes.Say("Shutting service-discovery-controller down"))
	})

	It("should return a http app json", func() {
		Eventually(session).Should(gbytes.Say("Server Started"))

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
		Eventually(session).Should(gbytes.Say("Server Started"))

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

	Context("when no nats server is provided in the config", func() {
		BeforeEach(func() {
			mainConfig.Nats = []config.NatsConfig{}
		})
		It("exits and print an error message", func() {
			Eventually(session).Should(gexec.Exit(2))
			Eventually(session).Should(gbytes.Say(
				"service-discovery.service-discovery-controller.*nats-connection-error.*no servers available for connection",
			))
		})
	})

	Context("when unable to connect to nats server", func() {
		BeforeEach(func() {
			natsRunner.Stop()
		})
		It("exits and print an error message", func() {
			Eventually(session).Should(gexec.Exit(2))
			Eventually(session).Should(gbytes.Say(
				"service-discovery.service-discovery-controller.*nats-connection-error.*no servers available for connection",
			))
		})
	})

	PContext("when nats bus registers a new app ", func() {
		var longApp *common.TestApp
		BeforeEach(func() {
			var err error

			mbusClient, err = newMessageBus(mainConfig)
			Expect(err).NotTo(HaveOccurred())

			requestMade := make(chan bool)
			requestProcessing := make(chan bool)

			longApp = common.NewTestApp([]string{"longapp.vcap.me"}, 1234, mbusClient, nil, "")
			longApp.AddHandler("/", func(w http.ResponseWriter, r *http.Request) {
				requestMade <- true
				<-requestProcessing
				_, ioErr := ioutil.ReadAll(r.Body)
				defer r.Body.Close()
				Expect(ioErr).ToNot(HaveOccurred())
				w.WriteHeader(http.StatusOK)
				w.Write([]byte{'b'})
			})
			longApp.Register()
			longApp.Listen()

		})
		PIt("registers the new route", func() {
			Eventually(session).Should(gbytes.Say("Server Started"))

			req, err := http.NewRequest("GET", "http://localhost:8055/v1/registration/app-id.internal.local.", nil)
			Expect(err).ToNot(HaveOccurred())
			resp, err := http.DefaultClient.Do(req)
			Expect(err).ToNot(HaveOccurred())
			respBody, err := ioutil.ReadAll(resp.Body)
			Expect(err).ToNot(HaveOccurred())
			fmt.Println(string(respBody))
			Expect(respBody).To(Equal(1))
			// routesUri := "http://localhost:8055/v1/registration/app-id.internal.local."
			// // routesUri := fmt.Sprintf("http://%s:%s@%s:%d/routes", config.Status.User, config.Status.Pass, localIP, statusPort)
			// Eventually(func() bool {
			// 	return appRegistered(routesUri, longApp)
			// }).Should(BeTrue())
		})
	})
})
