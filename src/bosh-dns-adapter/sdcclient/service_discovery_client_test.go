package sdcclient

import (
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

var _ = Describe("ServiceDiscoveryClient", func() {
	var (
		client             *ServiceDiscoveryClient
		fakeServer         *ghttp.Server
		fakeServerResponse http.HandlerFunc
	)

	JustBeforeEach(func() {
		fakeServer = ghttp.NewUnstartedServer()
		fakeServer.AppendHandlers(fakeServerResponse)
		fakeServer.HTTPTestServer.Start()
		client = NewServiceDiscoveryClient(fakeServer.URL())
	})

	AfterEach(func() {
		fakeServer.Close()
	})

	Context("when the server responds successfully", func() {
		BeforeEach(func() {
			fakeServerResponse = ghttp.CombineHandlers(
				ghttp.VerifyRequest("GET", "/v1/registration/app-id.sd-local.", ""),
				ghttp.RespondWith(http.StatusOK, `{
					"env": "",
					"Hosts": [
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

		It("returns the ips in the server response", func() {
			actualIPs, err := client.IPs("app-id.sd-local.")
			Expect(err).ToNot(HaveOccurred())

			Expect(actualIPs).To(Equal([]string{"192.168.0.1", "192.168.0.2"}))
		})
	})

	Context("when the server responds with malformed JSON", func() {
		BeforeEach(func() {
			fakeServerResponse = ghttp.CombineHandlers(
				ghttp.VerifyRequest("GET", "/v1/registration/app-id.sd-local.", ""),
				ghttp.RespondWith(http.StatusOK, `garbage`))
		})

		It("returns an error", func() {
			_, err := client.IPs("app-id.sd-local.")
			Expect(err).To(HaveOccurred())
		})
	})

	Context("when the server responds a non-200 response", func() {
		BeforeEach(func() {
			fakeServerResponse = ghttp.CombineHandlers(
				ghttp.VerifyRequest("GET", "/v1/registration/app-id.sd-local.", ""),
				ghttp.RespondWith(http.StatusBadRequest, `{}`))
		})

		It("returns an error", func() {
			_, err := client.IPs("app-id.sd-local.")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Received non successful response from server:"))
		})
	})
})
