package integration_test

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"testing"
	"time"

	"code.cloudfoundry.org/consuladapter/consulrunner"

	"service-discovery-controller/src/code.cloudfoundry.org/bbs"
	"service-discovery-controller/src/code.cloudfoundry.org/bbs/encryption"
	"service-discovery-controller/src/code.cloudfoundry.org/bbs/test_helpers"
	"service-discovery-controller/src/code.cloudfoundry.org/bbs/test_helpers/sqlrunner"

	"github.com/onsi/gomega/gexec"
	"github.com/onsi/gomega/ghttp"
)

var (
	emitterPath string

	oauthServer *ghttp.Server

	bbsPath    string
	bbsURL     *url.URL
	bbsConfig  bbsconfig.BBSConfig
	bbsRunner  *ginkgomon.Runner
	bbsProcess ifrit.Process

	consulRunner               *consulrunner.ClusterRunner
	bbsClient                  bbs.InternalClient
	logger                     *lagertest.TestLogger
	emitInterval, syncInterval time.Duration

	sqlProcess ifrit.Process
	sqlRunner  sqlrunner.SQLRunner
	bbsRunning = false
)

func TestIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Integration Suite")
}

var _ = SynchronizedBeforeSuite(func() []byte {
	bbs, err := gexec.Build("code.cloudfoundry.org/bbs/cmd/bbs", "-race")
	Expect(err).NotTo(HaveOccurred())

	serviceDiscoveryController, err := gexec.Build("service-discovery-controller/cmd/service-discovery-controller", "-race")
	Expect(err).NotTo(HaveOccurred())

	payload, err := json.Marshal(map[string]string{
		"bbs": bbs,
		"service-discovery-controller": serviceDiscoveryController,
	})

	Expect(err).NotTo(HaveOccurred())

	// var err error
	// certDir, err = ioutil.TempDir("", "netman-certs")
	// Expect(err).NotTo(HaveOccurred())
	//
	// certWriter, err := testsupport.NewCertWriter(certDir)
	// Expect(err).NotTo(HaveOccurred())
	//
	// paths.ServerCACertFile, err = certWriter.WriteCA("server-ca")
	// Expect(err).NotTo(HaveOccurred())
	// paths.ServerCertFile, paths.ServerKeyFile, err = certWriter.WriteAndSign("server", "server-ca")
	// Expect(err).NotTo(HaveOccurred())
	//
	// paths.ClientCACertFile, err = certWriter.WriteCA("client-ca")
	// Expect(err).NotTo(HaveOccurred())
	// paths.ClientCertFile, paths.ClientKeyFile, err = certWriter.WriteAndSign("client", "client-ca")
	// Expect(err).NotTo(HaveOccurred())
	//
	// fmt.Fprintf(GinkgoWriter, "building binary...")
	// paths.VxlanPolicyAgentPath, err = gexec.Build("vxlan-policy-agent/cmd/vxlan-policy-agent", "-race")
	// fmt.Fprintf(GinkgoWriter, "done")
	// Expect(err).NotTo(HaveOccurred())
	//
	// data, err := json.Marshal(paths)
	// Expect(err).NotTo(HaveOccurred())

	return payload
}, func(payload []byte) {
	oauthServer = startOAuthServer()

	binaries := map[string]string{}

	err := json.Unmarshal(payload, &binaries)
	Expect(err).NotTo(HaveOccurred())

	emitterPath = string(binaries["emitter"])

	dbName := fmt.Sprintf("service_discovery_%d", GinkgoParallelNode())
	sqlRunner = test_helpers.NewSQLRunner(dbName)

	consulRunner = consulrunner.NewClusterRunner(
		consulrunner.ClusterRunnerConfig{
			StartingPort: 9001 + GinkgoParallelNode()*consulrunner.PortOffsetLength,
			NumNodes:     1,
			Scheme:       "http",
		},
	)

	logger = lagertest.NewTestLogger("test")

	syncInterval = 200 * time.Millisecond
	emitInterval = time.Second

	bbsPath = string(binaries["bbs"])
	bbsPort := 13000 + GinkgoParallelNode()*2
	bbsHealthPort := bbsPort + 1
	bbsAddress := fmt.Sprintf("127.0.0.1:%d", bbsPort)
	bbsHealthAddress := fmt.Sprintf("127.0.0.1:%d", bbsHealthPort)
	routingAPIPath = string(binaries["routing-api"])

	bbsURL = &url.URL{
		Scheme: "http",
		Host:   bbsAddress,
	}

	bbsClient = bbs.NewClient(bbsURL.String())

	bbsConfig = bbsconfig.BBSConfig{
		ListenAddress:            bbsAddress,
		AdvertiseURL:             bbsURL.String(),
		AuctioneerAddress:        "http://some-address",
		DatabaseDriver:           sqlRunner.DriverName(),
		DatabaseConnectionString: sqlRunner.ConnectionString(),
		ConsulCluster:            consulRunner.ConsulCluster(),
		HealthAddress:            bbsHealthAddress,

		EncryptionConfig: encryption.EncryptionConfig{
			EncryptionKeys: map[string]string{"label": "key"},
			ActiveKeyLabel: "label",
		},
	}

	rand.Seed(ginkgoConfig.GinkgoConfig.RandomSeed + int64(GinkgoParallelNode()))
})

func startOAuthServer() *ghttp.Server {
	server := ghttp.NewUnstartedServer()
	tlsConfig, err := cfhttp.NewTLSConfig("fixtures/server.crt", "fixtures/server.key", "")
	Expect(err).NotTo(HaveOccurred())
	tlsConfig.ClientAuth = tls.NoClientCert

	server.HTTPTestServer.TLS = tlsConfig
	server.AllowUnhandledRequests = true
	server.UnhandledRequestStatusCode = http.StatusOK

	server.HTTPTestServer.StartTLS()

	publicKey := "-----BEGIN PUBLIC KEY-----\\n" +
		"MIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQDHFr+KICms+tuT1OXJwhCUmR2d\\n" +
		"KVy7psa8xzElSyzqx7oJyfJ1JZyOzToj9T5SfTIq396agbHJWVfYphNahvZ/7uMX\\n" +
		"qHxf+ZH9BL1gk9Y6kCnbM5R60gfwjyW1/dQPjOzn9N394zd2FJoFHwdq9Qs0wBug\\n" +
		"spULZVNRxq7veq/fzwIDAQAB\\n" +
		"-----END PUBLIC KEY-----"

	data := fmt.Sprintf("{\"alg\":\"rsa\", \"value\":\"%s\"}", publicKey)
	server.RouteToHandler("GET", "/token_key",
		ghttp.CombineHandlers(
			ghttp.VerifyRequest("GET", "/token_key"),
			ghttp.RespondWith(http.StatusOK, data)),
	)
	server.RouteToHandler("POST", "/oauth/token",
		ghttp.CombineHandlers(
			ghttp.VerifyBasicAuth("someclient", "somesecret"),
			func(w http.ResponseWriter, req *http.Request) {
				jsonBytes := []byte(`{"access_token":"some-token", "expires_in":10}`)
				w.Write(jsonBytes)
			}))

	return server
}
