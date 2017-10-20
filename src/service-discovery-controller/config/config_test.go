package config_test

import (
	. "service-discovery-controller/config"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Config", func() {
	Context("when created from valid JSON", func() {
		It("contains the values in the JSON", func() {
			configJSON := []byte(`{
				"address":"example.com",
				"port":"80053",
				"index":"62",
				"nats":[
					{
						"host": "a-nats-host",
						"port": 1,
						"user": "a-nats-user",
						"pass": "a-nats-pass"
					},
					{
						"host": "b-nats-host",
						"port": 2,
						"user": "b-nats-user",
						"pass": "b-nats-pass"
					}
				]
			}`)

			parsedConfig, err := NewConfig(configJSON)
			Expect(err).ToNot(HaveOccurred())

			Expect(parsedConfig.Address).To(Equal("example.com"))
			Expect(parsedConfig.Port).To(Equal("80053"))
			Expect(parsedConfig.Index).To(Equal("62"))
			Expect(parsedConfig.NatsServers()).To(ContainElement("nats://a-nats-user:a-nats-pass@a-nats-host:1"))
			Expect(parsedConfig.NatsServers()).To(ContainElement("nats://b-nats-user:b-nats-pass@b-nats-host:2"))
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
