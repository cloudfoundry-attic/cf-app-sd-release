package loglevel_test

import (
	. "service-discovery-controller/loglevel"

	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/tedsuo/ifrit"
	"net/http"
	"os"
	"service-discovery-controller/config"
	"strings"
	"time"
)

var _ = Describe("Server", func() {
	var (
		serverProc ifrit.Process
		testLogger *lagertest.TestLogger
		sink       *lager.ReconfigurableSink
		server     *Server
	)

	BeforeEach(func() {
		testLogger = lagertest.NewTestLogger("test")
		config := &config.Config{LogLevelPort: 8056, LogLevelAddress: "localhost"}
		sink = lager.NewReconfigurableSink(testLogger, lager.ERROR)

		server = NewServer(config, sink, testLogger)
	})

	Context("when it starts", func() {
		BeforeEach(func() {
			serverProc = ifrit.Invoke(server)
		})

		AfterEach(func() {
			serverProc.Signal(os.Interrupt)
			Eventually(serverProc.Wait()).Should(Receive())
		})

		It("sets the debug level", func() {
			response := requestLogChange("debug")
			Expect(response.StatusCode).To(Equal(http.StatusNoContent))
			Expect(testLogger.LogMessages()).To(ContainElement("test.Log level set to DEBUG"))
			Expect(sink.GetMinLevel()).To(Equal(lager.DEBUG))
		})

		It("sets the info level", func() {
			response := requestLogChange("info")
			Expect(response.StatusCode).To(Equal(http.StatusNoContent))
			Expect(testLogger.LogMessages()).To(ContainElement("test.Log level set to INFO"))
			Expect(sink.GetMinLevel()).To(Equal(lager.INFO))
		})

		It("rejects other levels", func() {
			response := requestLogChange("error")
			Expect(response.StatusCode).To(Equal(http.StatusBadRequest))
			Expect(sink.GetMinLevel()).To(Equal(lager.ERROR))
			Expect(testLogger.LogMessages()).To(ContainElement("test.Invalid log level requested: `error`. Skipping."))
		})
	})

	Context("when it is unable to start", func() {
		var conflictingServer *http.Server

		BeforeEach(func() {
			conflictingServer = launchConflictingServer()
		})

		AfterEach(func() {
			conflictingServer.Close()
			serverProc.Signal(os.Interrupt)
		})

		It("logs and quits", func() {
			serverProc = ifrit.Invoke(server)
			Eventually(serverProc.Wait()).Should(Receive())
			Eventually(testLogger.LogMessages(), 5*time.Second).Should(ContainElement("test.Listen and serve exited with error:"))
		})
	})
})

func launchConflictingServer() *http.Server {
	conflictingServer := &http.Server{Addr: "127.0.0.1:8056"}
	go func() { conflictingServer.ListenAndServe() }()
	client := &http.Client{}
	Eventually(func() bool {
		resp, _ := client.Get("http://127.0.0.1:8056")
		return resp.StatusCode == 404
	}).Should(BeTrue())
	return conflictingServer
}

func requestLogChange(logLevel string) *http.Response {
	client := &http.Client{}
	postBody := strings.NewReader(logLevel)
	response, err := client.Post("http://localhost:8056/log-level", "text/plain", postBody)
	Expect(err).ToNot(HaveOccurred())
	return response
}
