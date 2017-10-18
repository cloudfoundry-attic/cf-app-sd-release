package mbus

import (
	"encoding/json"

	"code.cloudfoundry.org/lager"
	"github.com/nats-io/nats"
	"github.com/pkg/errors"
)

type ServiceDiscoveryStartMessage struct {
	Id                               string `json:"id"`
	Host                             string `json:"host"`
	MinimumRegisterIntervalInSeconds int    `json:"minimumRegisterIntervalInSeconds"`
	PruneThresholdInSeconds          int    `json:"pruneThresholdInSeconds"`
}

type SubscriberOpts struct {
	ID                               string
	MinimumRegisterIntervalInSeconds int
	PruneThresholdInSeconds          int
	AcceptTLS                        bool
}

type RegistryMessage struct {
	Host string   `json:"host"`
	URIs []string `json:"uris"`
}

//go:generate counterfeiter -o fakes/address_table.go --fake-name AddressTable . AddressTable
type AddressTable interface {
	Add(hostnames []string, ip string)
	Remove(hostnames []string, ip string)
}

type Subscriber struct {
	natsClient NatsConn
	subOpts    SubscriberOpts
	table      AddressTable
	logger     lager.Logger
	localIP    string
}

//go:generate counterfeiter -o fakes/nats_conn.go --fake-name NatsConn . NatsConn
type NatsConn interface {
	PublishMsg(m *nats.Msg) error
	Close()
	Flush() error
	Subscribe(string, nats.MsgHandler) (*nats.Subscription, error)
}

//go:generate counterfeiter -o fakes/nats_conn_provider.go --fake-name NatsConnProvider . NatsConnProvider
type NatsConnProvider interface {
	Connection(opts ...nats.Option) (NatsConn, error)
}

//go:generate counterfeiter -o fakes/local_ip.go --fake-name LocalIP . LocalIP
type LocalIP interface {
	LocalIP() (string, error)
}

func NewSubscriber(
	natsConnBuilder NatsConnProvider,
	subOpts SubscriberOpts,
	table AddressTable,
	localIP LocalIP,
	logger lager.Logger,
) *Subscriber {

	ip, err := localIP.LocalIP()
	if err != nil {
		panic(err)
	}
	subscriber := &Subscriber{
		subOpts: subOpts,
		table:   table,
		logger:  logger,
		localIP: ip,
	}

	natsClient, err := natsConnBuilder.Connection(
		nats.ReconnectHandler(nats.ConnHandler(func(*nats.Conn) {
			subscriber.SendStartMessage()
		})),
	)

	if err != nil {
		panic(err)
	}

	subscriber.natsClient = natsClient
	return subscriber
}

func (s *Subscriber) SendStartMessage() error {
	discoveryMessageJson := s.mapSubOpts()

	msg := &nats.Msg{
		Subject: "service-discovery.start",
		Data:    discoveryMessageJson,
	}

	err := s.natsClient.PublishMsg(msg)
	if err != nil {
		return errors.Wrap(err, "unable to publish a start message")
	}

	return nil
}

func (s *Subscriber) Close() {
	s.natsClient.Close()
}

func (s *Subscriber) SetupGreetMsgHandler() error {
	discoveryMessageJson := s.mapSubOpts()

	_, err := s.natsClient.Subscribe("service-discovery.greet", nats.MsgHandler(func(greetMsg *nats.Msg) {
		msg := &nats.Msg{
			Subject: greetMsg.Reply,
			Data:    discoveryMessageJson,
		}

		_ = s.natsClient.PublishMsg(msg)
	}))

	if err != nil {
		return errors.Wrap(err, "unable to subscribe to greet messages")
	}

	err = s.natsClient.Flush()
	if err != nil {
		return errors.Wrap(err, "unable to flush subscribe greet message")
	}

	return nil
}

func (s *Subscriber) SetupAddressMessageHandler() {
	s.natsClient.Subscribe("service-discovery.register", nats.MsgHandler(func(msg *nats.Msg) {
		registryMessage := &RegistryMessage{}
		err := json.Unmarshal(msg.Data, registryMessage)
		if err != nil || registryMessage.Host == "" || len(registryMessage.URIs) == 0 {
			s.logger.Info("SetupAddressMessageHandler received a malformed message", lager.Data(map[string]interface{}{
				"msgJson": string(msg.Data),
			}))
			return
		}
		s.table.Add(registryMessage.URIs, registryMessage.Host)
	}))

	s.natsClient.Subscribe("service-discovery.unregister", nats.MsgHandler(func(msg *nats.Msg) {
		registryMessage := &RegistryMessage{}
		err := json.Unmarshal(msg.Data, registryMessage)
		if err != nil || len(registryMessage.URIs) == 0 {
			s.logger.Info("SetupAddressMessageHandler received a malformed message", lager.Data(map[string]interface{}{
				"msgJson": string(msg.Data),
			}))
			return
		}
		s.table.Remove(registryMessage.URIs, registryMessage.Host)
	}))
}

func (s *Subscriber) mapSubOpts() []byte {
	discoveryStartMessage := ServiceDiscoveryStartMessage{
		Id:   s.subOpts.ID,
		Host: s.localIP,
		MinimumRegisterIntervalInSeconds: s.subOpts.MinimumRegisterIntervalInSeconds,
		PruneThresholdInSeconds:          s.subOpts.PruneThresholdInSeconds,
	}

	discoveryMessageJson, _ := json.Marshal(discoveryStartMessage) //err should never happen

	return discoveryMessageJson
}
