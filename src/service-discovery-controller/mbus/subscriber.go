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

type Subscriber struct {
	NatsClient NatsConn
	SubOpts    SubscriberOpts
}

//go:generate counterfeiter -o fakes/nats_conn.go --fake-name NatsConn . NatsConn
type NatsConn interface {
	PublishMsg(m *nats.Msg) error
	Close()
	Flush() error
	Subscribe(string, nats.MsgHandler) (*nats.Subscription, error)
}

func NewSubscriber(natsClient NatsConn, subOpts SubscriberOpts) *Subscriber {
	return &Subscriber{
		natsClient,
		subOpts,
	}
}

func (s *Subscriber) SendStartMessage(host string) error {
	discoveryMessageJson := s.mapSubOpts(host)

	msg := &nats.Msg{
		Subject: "service-discovery.start",
		Data:    discoveryMessageJson,
	}

	err := s.NatsClient.PublishMsg(msg)
	if err != nil {
		return errors.Wrap(err, "unable to publish a start message")
	}

	return nil
}

func (s *Subscriber) Close() {
	s.NatsClient.Close()
}

func (s *Subscriber) SetupGreetMsgHandler(host string) error {
	discoveryMessageJson := s.mapSubOpts(host)

	_, err := s.NatsClient.Subscribe("service-discovery.greet", nats.MsgHandler(func(greetMsg *nats.Msg) {
		msg := &nats.Msg{
			Subject: greetMsg.Reply,
			Data:    discoveryMessageJson,
		}

		_ = s.NatsClient.PublishMsg(msg)
	}))

	if err != nil {
		return errors.Wrap(err, "unable to subscribe to greet messages")
	}

	err = s.NatsClient.Flush()
	if err != nil {
		return errors.Wrap(err, "unable to flush subscribe greet message")
	}

	return nil
}

func (s *Subscriber) mapSubOpts(host string) []byte {
	discoveryStartMessage := ServiceDiscoveryStartMessage{
		Id:                               s.SubOpts.ID,
		Host:                             host,
		MinimumRegisterIntervalInSeconds: s.SubOpts.MinimumRegisterIntervalInSeconds,
		PruneThresholdInSeconds:          s.SubOpts.PruneThresholdInSeconds,
	}

	discoveryMessageJson, _ := json.Marshal(discoveryStartMessage) //err should never happen

	return discoveryMessageJson
}
