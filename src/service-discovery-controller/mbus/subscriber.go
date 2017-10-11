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
	NatsClient *nats.Conn
	SubOpts    SubscriberOpts
}

func NewSubscriber(natsClient *nats.Conn, subOpts SubscriberOpts) *Subscriber {
	return &Subscriber{
		natsClient,
		subOpts,
	}
}

func (s *Subscriber) SendStartMessage(host string) error {
	discoveryStartMessage := s.mapSubOpts(host)

	discoveryMessageJson, err := json.Marshal(discoveryStartMessage)
	if err != nil {
		panic(err)
	}

	msg := &nats.Msg{
		Subject: "service-discovery.start",
		Data:    discoveryMessageJson,
	}

	err = s.NatsClient.PublishMsg(msg)
	if err != nil {
		return errors.Wrap(err, "unable to publish a start message")
	}

	return nil
}

func (s *Subscriber) Close() {
	//TODO: unsubscribe subscriptions?
	s.NatsClient.Close()
}

func (s *Subscriber) SendGreetMessage(host string) error {
	discoveryStartMessage := s.mapSubOpts(host)

	discoveryMessageJson, err := json.Marshal(discoveryStartMessage)
	if err != nil {
		panic(err)
	}

	_, err = s.NatsClient.Subscribe("service-discovery.greet", nats.MsgHandler(func(greetMsg *nats.Msg) {
		msg := &nats.Msg{
			Subject: greetMsg.Reply,
			Data:    discoveryMessageJson,
		}

		err = s.NatsClient.PublishMsg(msg)
		if err != nil {
			panic(err)
		}

		err = s.NatsClient.Flush()
		if err != nil {
			panic(err)
		}
	}))

	if err != nil {
		return errors.Wrap(err, "unable to subscribe to greet messages")
	}

	err = s.NatsClient.Flush()
	if err != nil {
		panic(err)
	}

	return nil
}

func (s *Subscriber) mapSubOpts(host string) ServiceDiscoveryStartMessage {
	return ServiceDiscoveryStartMessage{
		Id:   s.SubOpts.ID,
		Host: host,
		MinimumRegisterIntervalInSeconds: s.SubOpts.MinimumRegisterIntervalInSeconds,
		PruneThresholdInSeconds:          s.SubOpts.PruneThresholdInSeconds,
	}
}
