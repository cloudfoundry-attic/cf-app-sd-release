package mbus

import (
	"encoding/json"

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
}

type Subscriber struct {
	natsClient NatsConn
	subOpts    SubscriberOpts
	table      AddressTable
}

//go:generate counterfeiter -o fakes/nats_conn.go --fake-name NatsConn . NatsConn
type NatsConn interface {
	PublishMsg(m *nats.Msg) error
	Close()
	Flush() error
	Subscribe(string, nats.MsgHandler) (*nats.Subscription, error)
}

func NewSubscriber(natsClient NatsConn, subOpts SubscriberOpts, table AddressTable) *Subscriber {
	return &Subscriber{
		natsClient,
		subOpts,
		table,
	}
}

func (s *Subscriber) SendStartMessage(host string) error {
	discoveryMessageJson := s.mapSubOpts(host)

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

func (s *Subscriber) SetupGreetMsgHandler(host string) error {
	discoveryMessageJson := s.mapSubOpts(host)

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
		json.Unmarshal(msg.Data, registryMessage)
		s.table.Add(registryMessage.URIs, registryMessage.Host)
	}))
}

func (s *Subscriber) mapSubOpts(host string) []byte {
	discoveryStartMessage := ServiceDiscoveryStartMessage{
		Id:                               s.subOpts.ID,
		Host:                             host,
		MinimumRegisterIntervalInSeconds: s.subOpts.MinimumRegisterIntervalInSeconds,
		PruneThresholdInSeconds:          s.subOpts.PruneThresholdInSeconds,
	}

	discoveryMessageJson, _ := json.Marshal(discoveryStartMessage) //err should never happen

	return discoveryMessageJson
}
