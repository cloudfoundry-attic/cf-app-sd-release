package config_test

import (
	. "service-discovery-controller/config"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Config", func() {
	Describe("NewConfig", func() {
		Context("when created from valid JSON", func() {
			It("contains the values in the JSON", func() {
				configJSON := []byte(`{
				"address":"example.com",
				"port":"80053",
				"log_level":"debug",
				"nats": [{
					"host": "host",
					"port": 1234,
					"user": "user",
					"pass": "pass"
				}]
			}`)

				parsedConfig, err := NewConfig(configJSON)
				Expect(err).ToNot(HaveOccurred())

				Expect(parsedConfig.Address).To(Equal("example.com"))
				Expect(parsedConfig.Port).To(Equal("80053"))
				Expect(parsedConfig.LogLevel).To(Equal("debug"))

				Expect(parsedConfig.Nats).To(HaveLen(1))
				Expect(parsedConfig.Nats[0].Host).To(Equal("host"))
				Expect(parsedConfig.Nats[0].Port).To(Equal(uint16(1234)))
				Expect(parsedConfig.Nats[0].User).To(Equal("user"))
				Expect(parsedConfig.Nats[0].Pass).To(Equal("pass"))
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

	Describe("NatsServers", func() {
		var config *Config
		BeforeEach(func() {
			configJSON := []byte(`{
				"address":"example.com",
				"port":"80053",
				"log_level":"debug",
				"nats": [{
					"host": "host",
					"port": 1234,
					"user": "user",
					"pass": "pass"
				}]
			}`)

			var err error
			config, err = NewConfig(configJSON)
			Expect(err).ToNot(HaveOccurred())
		})
		It("returns the Nats server addresses", func() {
			Expect(config.NatsServers()).To(Equal([]string{"nats://user:pass@host:1234"}))
		})
	})

})
