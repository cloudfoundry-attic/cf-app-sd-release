package mbus_test

import (
	. "service-discovery-controller/mbus"

	"encoding/json"

	"time"

	"github.com/nats-io/gnatsd/server"
	"github.com/nats-io/nats"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/st3v/glager"
	"service-discovery-controller/mbus/fakes"
	"github.com/pkg/errors"
	"code.cloudfoundry.org/lager"
)

var _ = Describe("Subscriber", func() {
	var (
		gnatsServer      *server.Server
		fakeRouteEmitter *nats.Conn
		subscriber       *Subscriber
		subOpts          SubscriberOpts
		natsUrl          string
		addressTable     *fakes.AddressTable
		subcriberLogger  lager.Logger
	)

	BeforeEach(func() {
		gnatsServer = RunServerOnPort(8080)
		gnatsServer.Start()

		natsUrl = "nats://" + gnatsServer.Addr().String()
		fakeRouteEmitter = getNatsClient(natsUrl)

		subOpts = SubscriberOpts{
			ID:                               "Fake-Subscriber-ID",
			MinimumRegisterIntervalInSeconds: 60,
			PruneThresholdInSeconds:          120,
		}
		natsClient, err := nats.Connect(natsUrl)
		Expect(err).ToNot(HaveOccurred())

		addressTable = &fakes.AddressTable{}
		subcriberLogger = NewLogger("test")

		subscriber = NewSubscriber(natsClient, subOpts, addressTable, subcriberLogger)
	})

	AfterEach(func() {
		subscriber.Close()
		fakeRouteEmitter.Close()
		gnatsServer.Shutdown()
	})

	It("sends a start message", func() {
		msgChan := make(chan *nats.Msg, 1)

		_, err := fakeRouteEmitter.ChanSubscribe("service-discovery.start", msgChan)
		Expect(err).ToNot(HaveOccurred())
		Expect(fakeRouteEmitter.Flush()).To(Succeed())

		err = subscriber.SendStartMessage("127.0.0.1:8080")
		Expect(err).ToNot(HaveOccurred())

		var msg *nats.Msg
		var serviceDiscoveryData ServiceDiscoveryStartMessage

		Eventually(msgChan, 4).Should(Receive(&msg))

		Expect(msg).ToNot(BeNil())

		err = json.Unmarshal(msg.Data, &serviceDiscoveryData)
		Expect(err).ToNot(HaveOccurred())

		Expect(serviceDiscoveryData.Id).To(Equal(subOpts.ID))
		Expect(serviceDiscoveryData.MinimumRegisterIntervalInSeconds).To(Equal(subOpts.MinimumRegisterIntervalInSeconds))
		Expect(serviceDiscoveryData.PruneThresholdInSeconds).To(Equal(subOpts.PruneThresholdInSeconds))
		Expect(serviceDiscoveryData.Host).ToNot(BeEmpty())
	})

	It("when a greeting message is received it responds", func() {
		msgChan := make(chan *nats.Msg, 1)

		_, err := fakeRouteEmitter.ChanSubscribe("service-discovery.greet.test.response", msgChan)
		Expect(err).ToNot(HaveOccurred())
		Expect(fakeRouteEmitter.Flush()).To(Succeed())

		time.Sleep(1 * time.Second)

		err = subscriber.SetupGreetMsgHandler("127.0.0.1:8080")
		Expect(err).NotTo(HaveOccurred())

		Expect(fakeRouteEmitter.PublishRequest("service-discovery.greet", "service-discovery.greet.test.response", []byte{})).To(Succeed())
		Expect(fakeRouteEmitter.Flush()).To(Succeed())

		var msg *nats.Msg
		var serviceDiscoveryData ServiceDiscoveryStartMessage
		Eventually(msgChan, 10*time.Second).Should(Receive(&msg))
		Expect(msg).ToNot(BeNil())

		err = json.Unmarshal(msg.Data, &serviceDiscoveryData)
		Expect(err).ToNot(HaveOccurred())

		Expect(serviceDiscoveryData.Id).To(Equal(subOpts.ID))
		Expect(serviceDiscoveryData.MinimumRegisterIntervalInSeconds).To(Equal(subOpts.MinimumRegisterIntervalInSeconds))
		Expect(serviceDiscoveryData.PruneThresholdInSeconds).To(Equal(subOpts.PruneThresholdInSeconds))
		Expect(serviceDiscoveryData.Host).ToNot(BeEmpty())
	})

	Context("when a greeting message for a non-default subject is sent", func() {
		It("it responds", func() {
			msgChan := make(chan *nats.Msg, 1)

			_, err := fakeRouteEmitter.ChanSubscribe("service-discovery.greet-1.test.response", msgChan)
			Expect(err).ToNot(HaveOccurred())
			Expect(fakeRouteEmitter.Flush()).To(Succeed())

			err = subscriber.SetupGreetMsgHandler("127.0.0.1:8080")
			Expect(err).NotTo(HaveOccurred())

			err = fakeRouteEmitter.PublishRequest("service-discovery.greet", "service-discovery.greet-1.test.response", []byte{})
			Expect(err).ToNot(HaveOccurred())
			Expect(fakeRouteEmitter.Flush()).To(Succeed())

			var msg *nats.Msg
			var serviceDiscoveryData ServiceDiscoveryStartMessage
			Eventually(msgChan, 4*time.Second).Should(Receive(&msg))
			Expect(msg).ToNot(BeNil())

			err = json.Unmarshal(msg.Data, &serviceDiscoveryData)
			Expect(err).ToNot(HaveOccurred())

			Expect(serviceDiscoveryData.Id).To(Equal(subOpts.ID))
			Expect(serviceDiscoveryData.MinimumRegisterIntervalInSeconds).To(Equal(subOpts.MinimumRegisterIntervalInSeconds))
			Expect(serviceDiscoveryData.PruneThresholdInSeconds).To(Equal(subOpts.PruneThresholdInSeconds))
			Expect(serviceDiscoveryData.Host).ToNot(BeEmpty())
		})
	})

	Context("when nats client connection is closed", func() {
		BeforeEach(func() {
			subscriber.Close()
		})

		It("and fails to publish message", func() {
			err := subscriber.SendStartMessage("127.0.0.1:8080")
			Expect(err).To(HaveOccurred())

			Expect(err.Error()).To(Equal("unable to publish a start message: nats: connection closed"))
		})

		It("and fails to subscribe to greet messages", func() {
			err := subscriber.SetupGreetMsgHandler("127.0.0.1:8080")
			Expect(err).To(HaveOccurred())

			Expect(err.Error()).To(Equal("unable to subscribe to greet messages: nats: connection closed"))
		})
	})

	Context("when subscriber loses nats server connectivity and then regains connectivity", func() {
		It("should still be able to send a start message", func() {
			msgChan := make(chan *nats.Msg, 1)
			_, err := fakeRouteEmitter.ChanSubscribe("service-discovery.start", msgChan)
			Expect(err).ToNot(HaveOccurred())
			Expect(fakeRouteEmitter.Flush()).To(Succeed())

			By("gnatsd server stops", func() {
				gnatsServer.Shutdown()
			})

			By("gnatsd starts back up", func() {
				gnatsServer = RunServerOnPort(8080)
				gnatsServer.Start()
			})

			Eventually(func() bool {
				return fakeRouteEmitter.IsConnected()
			}, 10*time.Second).Should(BeTrue())

			err = subscriber.SendStartMessage("127.0.0.1:8080")
			Expect(err).ToNot(HaveOccurred())

			var msg *nats.Msg
			Eventually(msgChan, 4).ShouldNot(Receive(&msg))

			var serviceDiscoveryData ServiceDiscoveryStartMessage
			Eventually(msgChan, 30*time.Second).Should(Receive(&msg))

			Expect(msg).ToNot(BeNil())

			err = json.Unmarshal(msg.Data, &serviceDiscoveryData)
			Expect(err).ToNot(HaveOccurred())

			Expect(serviceDiscoveryData.Id).To(Equal(subOpts.ID))
			Expect(serviceDiscoveryData.MinimumRegisterIntervalInSeconds).To(Equal(subOpts.MinimumRegisterIntervalInSeconds))
			Expect(serviceDiscoveryData.PruneThresholdInSeconds).To(Equal(subOpts.PruneThresholdInSeconds))
			Expect(serviceDiscoveryData.Host).ToNot(BeEmpty())
		})
	})

	Context("when subscriber loses nats server connectivity", func() {
		BeforeEach(func() {
			natsClient, err := nats.Connect(natsUrl, nats.Option(func(options *nats.Options) error {
				options.ReconnectWait = 1 * time.Millisecond
				return nil
			}))
			Expect(err).ToNot(HaveOccurred())
			subscriber = NewSubscriber(natsClient, subOpts, addressTable, subcriberLogger)
		})

		It("sending a start message should return an error", func() {
			gnatsServer.Shutdown()

			Eventually(func() error {
				return subscriber.SendStartMessage("127.0.0.1:8080")
			}).ShouldNot(Succeed())
		})
	})

	Context("when a registration message is received", func() {
		It("should write it to the address table", func() {
			subscriber.SetupAddressMessageHandler()

			natsRegistryMsg := nats.Msg{
				Subject: "service-discovery.register",
				Data: []byte(`{
					"host": "192.168.0.1",
					"uris": ["foo.com", "0.foo.com"]
				}`),
			}

			Eventually(func() int {
				fakeRouteEmitter.PublishMsg(&natsRegistryMsg)
				return addressTable.AddCallCount()
			}).Should(Equal(1))

			hostnames, ip := addressTable.AddArgsForCall(0)

			Expect(hostnames).To(Equal([]string{"foo.com", "0.foo.com"}))
			Expect(ip).To(Equal("192.168.0.1"))
		})

		Context("when the message is malformed", func() {
			It("should not add the garbage", func() {
				subscriber.SetupAddressMessageHandler()

				json := `garbage "0.foo.com"] }`
				natsRegistryMsg := nats.Msg{
					Subject: "service-discovery.register",
					Data:    []byte(json),
				}

				Eventually(func() lager.Logger {
					fakeRouteEmitter.PublishMsg(&natsRegistryMsg)
					return subcriberLogger
				}).Should(HaveLogged(
					Info(
						Message("test.SetupAddressMessageHandler received a malformed message"),
						Data("msgJson", json),
					)))

				Expect(addressTable.AddCallCount()).To(Equal(0))
			})
		})

		Context("when a registration message does not contain host info", func() {
			It("should not add", func() {
				subscriber.SetupAddressMessageHandler()

				json := `{
					"uris": ["foo.com", "0.foo.com"]
				}`
				natsRegistryMsg := nats.Msg{
					Subject: "service-discovery.register",
					Data:    []byte(json),
				}

				Eventually(func() lager.Logger {
					fakeRouteEmitter.PublishMsg(&natsRegistryMsg)
					return subcriberLogger
				}).Should(HaveLogged(
					Info(
						Message("test.SetupAddressMessageHandler received a malformed message"),
						Data("msgJson", json),
					)))

				Expect(addressTable.AddCallCount()).To(Equal(0))
			})
		})

		Context("when a registration message does not contain URIS", func() {
			It("should not add", func() {
				subscriber.SetupAddressMessageHandler()

				json := `{
									"host": "192.168.0.1"
				}`
				natsRegistryMsg := nats.Msg{
					Subject: "service-discovery.register",
					Data:    []byte(json),
				}

				Eventually(func() lager.Logger {
					fakeRouteEmitter.PublishMsg(&natsRegistryMsg)
					return subcriberLogger
				}).Should(HaveLogged(
					Info(
						Message("test.SetupAddressMessageHandler received a malformed message"),
						Data("msgJson", json),
					)))

				Expect(addressTable.AddCallCount()).To(Equal(0))
			})
		})
	})

	Describe("Edge error cases", func() {
		var fakeNatsConn *fakes.NatsConn
		BeforeEach(func() {
			fakeNatsConn = &fakes.NatsConn{}
			subscriber = NewSubscriber(fakeNatsConn, subOpts, addressTable, subcriberLogger)
		})

		Context("when sending a greet message and fails to flush", func() {
			BeforeEach(func() {
				fakeNatsConn.FlushReturns(errors.New("failed to flush"))
			})

			It("should return an error", func() {
				Expect(subscriber.SetupGreetMsgHandler("fake-host")).To(MatchError("unable to flush subscribe greet message: failed to flush"))
			})
		})
	})
})

func getNatsClient(natsUrl string) *nats.Conn {
	natsClient, err := nats.Connect(natsUrl)
	Expect(err).NotTo(HaveOccurred())
	return natsClient
}
