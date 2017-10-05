package config_test

import (
	. "bosh-dns-adapter/config"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Config", func() {
	Context("when created from valid JSON", func() {
		It("contains the values in the JSON", func() {
			configJSON := []byte(`{
				"address":"example.com",
				"port":"80053",
				"service_discovery_controller_address":"bar.com",
				"service_discovery_controller_port":"80055"
			}`)

			parsedConfig, err := NewConfig(configJSON)
			Expect(err).ToNot(HaveOccurred())

			Expect(parsedConfig.Address).To(Equal("example.com"))
			Expect(parsedConfig.Port).To(Equal("80053"))
			Expect(parsedConfig.ServiceDiscoveryControllerAddress).To(Equal("bar.com"))
			Expect(parsedConfig.ServiceDiscoveryControllerPort).To(Equal("80055"))
		})
	})

	Context("when constructed with invalid JSON", func() {
		It("returns an error", func() {
			configJSON := []byte(`garbage`)
			_, err := NewConfig(configJSON)
			Expect(err).To(HaveOccurred())
		})
	})
})
