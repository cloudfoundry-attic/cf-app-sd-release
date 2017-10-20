package mbus_test

import (
	. "service-discovery-controller/mbus"

	"encoding/json"

	"time"

	"service-discovery-controller/mbus/fakes"

	"code.cloudfoundry.org/lager"
	"github.com/nats-io/gnatsd/server"
	"github.com/nats-io/nats"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	. "github.com/st3v/glager"
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
		localIP          string
		startMsgChan     chan *nats.Msg
		greetMsgChan     chan *nats.Msg
	)

	BeforeEach(func() {
		gnatsServer = RunServerOnPort(8080)
		gnatsServer.Start()

		natsUrl = "nats://username:password@" + gnatsServer.Addr().String()
		fakeRouteEmitter = newFakeRouteEmitter(natsUrl)

		startMsgChan = make(chan *nats.Msg, 1)
		_, err := fakeRouteEmitter.ChanSubscribe("service-discovery.start", startMsgChan)
		Expect(err).ToNot(HaveOccurred())

		greetMsgChan = make(chan *nats.Msg, 1)

		_, err = fakeRouteEmitter.ChanSubscribe("service-discovery.greet.test.response", greetMsgChan)
		Expect(err).ToNot(HaveOccurred())

		Expect(fakeRouteEmitter.Flush()).To(Succeed())

		subOpts = SubscriberOpts{
			ID: "Fake-Subscriber-ID",
			MinimumRegisterIntervalInSeconds: 60,
			PruneThresholdInSeconds:          120,
		}

		addressTable = &fakes.AddressTable{}
		subcriberLogger = NewLogger("test")

		provider := &NatsConnWithUrlProvider{Url: natsUrl}

		localIP = "192.168.0.1"

		subscriber = NewSubscriber(provider, subOpts, addressTable, localIP, subcriberLogger)
		Expect(subscriber.Run()).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		subscriber.Close()
		fakeRouteEmitter.Close()
		gnatsServer.Shutdown()
	})

	It("sends a start message", func() {
		var msg *nats.Msg
		var serviceDiscoveryData ServiceDiscoveryStartMessage

		Eventually(startMsgChan, 4).Should(Receive(&msg))

		Expect(msg).ToNot(BeNil())

		err := json.Unmarshal(msg.Data, &serviceDiscoveryData)
		Expect(err).ToNot(HaveOccurred())

		Expect(serviceDiscoveryData.Id).To(Equal(subOpts.ID))
		Expect(serviceDiscoveryData.MinimumRegisterIntervalInSeconds).To(Equal(subOpts.MinimumRegisterIntervalInSeconds))
		Expect(serviceDiscoveryData.PruneThresholdInSeconds).To(Equal(subOpts.PruneThresholdInSeconds))
		Expect(serviceDiscoveryData.Host).ToNot(BeEmpty())
	})

	It("when a greeting message is received it responds", func() {
		Expect(fakeRouteEmitter.PublishRequest("service-discovery.greet", "service-discovery.greet.test.response", []byte{})).To(Succeed())
		Expect(fakeRouteEmitter.Flush()).To(Succeed())

		var msg *nats.Msg
		var serviceDiscoveryData ServiceDiscoveryStartMessage
		Eventually(greetMsgChan, 10*time.Second).Should(Receive(&msg))
		Expect(msg).ToNot(BeNil())

		err := json.Unmarshal(msg.Data, &serviceDiscoveryData)
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

		It("logs a message", func() {
			Eventually(subcriberLogger).Should(HaveLogged(
				Info(
					Message("test.ClosedHandler unexpected close of nats connection"),
					Data("last_error", nil),
				)))
		})
	})

	Context("when the nats server stops", func() {
		It("logs a message", func() {
			By("gnatsd server stops", func() {
				gnatsServer.Shutdown()
			})

			Eventually(subcriberLogger, 5*time.Second).Should(HaveLogged(
				Info(
					Message("test.DisconnectHandler disconnected from nats server"),
					Data("last_error", nil),
				)))
		})
	})

	Context("when subscriber loses nats server connectivity and then regains connectivity", func() {
		It("should send a start message", func() {
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
			Expect(serviceDiscoveryData.Host).To(Equal("192.168.0.1"))

			Eventually(subcriberLogger, 5*time.Second).Should(HaveLogged(
				Info(
					Message("test.ReconnectHandler reconnected to nats server"),
					Data("nats_host", "nats://"+gnatsServer.Addr().String()),
				)))

		})
	})

	Context("when a registration message is received", func() {
		It("should write it to the address table", func() {
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
						Message("test.AddressMessageHandler received a malformed register message"),
						Data("msgJson", json),
					)))

				Expect(addressTable.AddCallCount()).To(Equal(0))
			})
		})

		Context("when a registration message does not contain host info", func() {
			It("should not add", func() {
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
						Message("test.AddressMessageHandler received a malformed register message"),
						Data("msgJson", json),
					)))

				Expect(addressTable.AddCallCount()).To(Equal(0))
			})
		})

		Context("when a registration message does not contain URIS", func() {
			It("should not add", func() {
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
						Message("test.AddressMessageHandler received a malformed register message"),
						Data("msgJson", json),
					)))

				Expect(addressTable.AddCallCount()).To(Equal(0))
			})
		})
	})

	Context("when an unregister message is received", func() {
		It("should remove it from the address table", func() {
			natsUnRegisterMsg := nats.Msg{
				Subject: "service-discovery.unregister",
				Data: []byte(`{
					"host": "192.168.0.1",
					"uris": ["foo.com", "0.foo.com"]
				}`),
			}

			Eventually(func() int {
				fakeRouteEmitter.PublishMsg(&natsUnRegisterMsg)
				return addressTable.RemoveCallCount()
			}).Should(Equal(1))

			uris, host := addressTable.RemoveArgsForCall(0)
			Expect(uris).To(Equal([]string{"foo.com", "0.foo.com"}))
			Expect(host).To(Equal("192.168.0.1"))
		})

		Context("when the message is malformed", func() {
			It("should not remove the garbage", func() {
				json := `garbage "0.foo.com"] }`
				natsUnRegisterMsg := nats.Msg{
					Subject: "service-discovery.unregister",
					Data:    []byte(json),
				}

				Eventually(func() lager.Logger {
					fakeRouteEmitter.PublishMsg(&natsUnRegisterMsg)
					return subcriberLogger
				}).Should(HaveLogged(
					Info(
						Message("test.AddressMessageHandler received a malformed unregister message"),
						Data("msgJson", json),
					)))

				Expect(addressTable.RemoveCallCount()).To(Equal(0))
			})
		})

		Context("when an unregister message does not contain host info", func() {
			It("should remove it from the address table", func() {
				json := `{
					"uris": ["foo.com", "0.foo.com"]
				}`
				natsUnRegisterMsg := nats.Msg{
					Subject: "service-discovery.unregister",
					Data:    []byte(json),
				}

				Eventually(func() int {
					fakeRouteEmitter.PublishMsg(&natsUnRegisterMsg)
					return addressTable.RemoveCallCount()
				}).Should(Equal(1))

				Expect(addressTable.RemoveArgsForCall(0)).To(Equal([]string{"foo.com", "0.foo.com"}))
			})
		})

		Context("when a registration message does not contain URIS", func() {
			It("should not remove and log", func() {
				json := `{ "host": "192.168.0.1" }`
				natsUnRegisterMsg := nats.Msg{
					Subject: "service-discovery.unregister",
					Data:    []byte(json),
				}

				Eventually(func() lager.Logger {
					fakeRouteEmitter.PublishMsg(&natsUnRegisterMsg)
					return subcriberLogger
				}).Should(HaveLogged(
					Info(
						Message("test.AddressMessageHandler received a malformed unregister message"),
						Data("msgJson", json),
					)))

				Expect(addressTable.RemoveCallCount()).To(Equal(0))
			})
		})
	})

	Describe("Edge error cases", func() {
		var (
			natsConn *fakes.NatsConn
			provider *fakes.NatsConnProvider
		)

		BeforeEach(func() {
			natsConn = &fakes.NatsConn{}
			provider = &fakes.NatsConnProvider{}
			provider.ConnectionReturns(natsConn, nil)

		})

		Context("when initializing the nats connection returns an error", func() {
			BeforeEach(func() {
				provider.ConnectionReturns(natsConn, errors.New("CANT"))

				subscriber.Close()
				subscriber = NewSubscriber(provider, subOpts, addressTable, localIP, subcriberLogger)
			})

			It("run returns an error", func() {
				err := subscriber.Run()
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError("unable to create nats connection: CANT"))
			})
		})

		Context("when calling run and sending start message fails", func() {
			BeforeEach(func() {
				natsConn.PublishMsgReturns(errors.New("NO START"))
				subscriber = NewSubscriber(provider, subOpts, addressTable, localIP, subcriberLogger)
			})

			It("returns an error", func() {
				err := subscriber.Run()
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError("unable to publish a start message: NO START"))
			})

			It("self closes", func() {
				subscriber.Run()
				Expect(natsConn.CloseCallCount()).To(Equal(1))
			})
		})

		Context("when calling run and sending greet message fails", func() {
			BeforeEach(func() {
				natsConn.PublishMsgReturnsOnCall(0, nil)
				natsConn.SubscribeReturns(nil, errors.New("NO GREET"))

				subscriber = NewSubscriber(provider, subOpts, addressTable, localIP, subcriberLogger)
			})

			It("self closes", func() {
				subscriber.Run()
				Expect(natsConn.CloseCallCount()).To(Equal(1))
			})

			It("returns an error", func() {
				err := subscriber.Run()
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError("NO GREET"))
			})

			It("logs an error", func() {
				err := subscriber.Run()
				Expect(err).To(HaveOccurred())

				Expect(subcriberLogger).To(HaveLogged(
					Error(
						err,
						Message("test.setupGreetMsgHandler unable to subscribe to greet messages"),
					)))
			})
		})

		Context("when calling run and subscribing to register fails", func() {
			BeforeEach(func() {
				natsConn.PublishMsgReturnsOnCall(0, nil)
				natsConn.SubscribeReturnsOnCall(1, nil, errors.New("NO SUBSCRIBE"))

				subscriber = NewSubscriber(provider, subOpts, addressTable, localIP, subcriberLogger)
			})

			It("returns an error", func() {
				err := subscriber.Run()
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError("NO SUBSCRIBE"))
			})

			It("self closes", func() {
				subscriber.Run()
				Expect(natsConn.CloseCallCount()).To(Equal(1))
			})

			It("logs an error", func() {
				err := subscriber.Run()
				Expect(err).To(HaveOccurred())

				Expect(subcriberLogger).To(HaveLogged(
					Error(
						err,
						Message("test.setupAddressMessageHandler unable to subscribe to service-discovery.register"),
					)))
			})
		})

		Context("when calling run and subscribing to unregister fails", func() {
			BeforeEach(func() {
				natsConn.PublishMsgReturnsOnCall(0, nil)
				natsConn.SubscribeReturnsOnCall(2, nil, errors.New("NO SUBSCRIBE when unregister"))

				subscriber = NewSubscriber(provider, subOpts, addressTable, localIP, subcriberLogger)
			})

			It("returns an error", func() {
				err := subscriber.Run()
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError("NO SUBSCRIBE when unregister"))
			})

			It("self closes", func() {
				subscriber.Run()
				Expect(natsConn.CloseCallCount()).To(Equal(1))
			})

			It("logs an error", func() {
				err := subscriber.Run()
				Expect(err).To(HaveOccurred())

				Expect(subcriberLogger).To(HaveLogged(
					Error(
						err,
						Message("test.setupAddressMessageHandler unable to subscribe to service-discovery.unregister"),
					)))
			})
		})

		Context("when sending a greet message and fails to flush", func() {
			BeforeEach(func() {
				provider.ConnectionReturns(natsConn, nil)

				subscriber.Close()
				subscriber = NewSubscriber(provider, subOpts, addressTable, localIP, subcriberLogger)
				natsConn.FlushReturns(errors.New("failed to flush"))
			})

			It("should return an error", func() {
				err := subscriber.Run()
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError("failed to flush"))
			})

			It("logs an error", func() {
				err := subscriber.Run()
				Expect(err).To(HaveOccurred())
				Expect(subcriberLogger).To(HaveLogged(
					Error(
						err,
						Message("test.setupGreetMsgHandler unable to flush subscribe greet message"),
					)))

			})
		})

		Context("when attempting to run subscriber more than once", func() {
			BeforeEach(func() {
				provider.ConnectionReturns(natsConn, nil)

				subscriber.Close()
				subscriber = NewSubscriber(provider, subOpts, addressTable, localIP, subcriberLogger)
			})

			It("should not have any side effects", func() {
				Expect(subscriber.Run()).To(Succeed())
				Expect(subscriber.Run()).To(Succeed())

				Expect(provider.ConnectionCallCount()).To(Equal(1))
			})
		})

	})
})

func newFakeRouteEmitter(natsUrl string) *nats.Conn {
	natsClient, err := nats.Connect(natsUrl, nats.ReconnectWait(1*time.Nanosecond))
	Expect(err).NotTo(HaveOccurred())
	return natsClient
}
