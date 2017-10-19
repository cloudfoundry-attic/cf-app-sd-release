package mbus

import (
	"encoding/json"

	"sync"

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
	IP         string   `json:"host"`
	InfraNames []string `json:"uris"`
}

//go:generate counterfeiter -o fakes/address_table.go --fake-name AddressTable . AddressTable
type AddressTable interface {
	Add(infraNames []string, ip string)
	Remove(infraNames []string, ip string)
}

type Subscriber struct {
	natsConnProvider NatsConnProvider
	subOpts          SubscriberOpts
	table            AddressTable
	logger           lager.Logger
	localIP          string
	natsClient       NatsConn
	once             sync.Once
}

//go:generate counterfeiter -o fakes/nats_conn.go --fake-name NatsConn . NatsConn
type NatsConn interface {
	PublishMsg(m *nats.Msg) error
	Close()
	Flush() error
	ConnectedUrl() string
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
	natsConnProvider NatsConnProvider,
	subOpts SubscriberOpts,
	table AddressTable,
	localIP string,
	logger lager.Logger,
) *Subscriber {
	return &Subscriber{
		natsConnProvider: natsConnProvider,
		subOpts:          subOpts,
		table:            table,
		logger:           logger,
		localIP:          localIP,
	}
}

func (s *Subscriber) Run() error {
	var err error
	s.once.Do(func() {
		var natsClient NatsConn
		natsClient, err = s.natsConnProvider.Connection(
			nats.ReconnectHandler(nats.ConnHandler(func(conn *nats.Conn) {
				s.logger.Info(
					"ReconnectHandler reconnected to nats server",
					lager.Data{"nats_host": conn.ConnectedUrl()}, // TODO: user pass santization
				)
				s.sendStartMessage()
			})),
			nats.DisconnectHandler(nats.ConnHandler(func(conn *nats.Conn) {
				s.logger.Info(
					"DisconnectHandler disconnected from nats server",
					lager.Data{"last_error": conn.LastError()},
				)
			})),
			nats.ClosedHandler(nats.ConnHandler(func(conn *nats.Conn) {
				s.logger.Info(
					"ClosedHandler unexpected close of nats connection",
					lager.Data{"last_error": conn.LastError()},
				)
			})),
		)

		if err != nil {
			err = errors.Wrap(err, "unable to create nats connection")
			return
		}

		s.natsClient = natsClient

		s.logger.Info(
			"Connected to NATS server",
			lager.Data{"nats_host": natsClient.ConnectedUrl()}, // TODO: user pass santization
		)

		err = s.sendStartMessage()
		if err != nil {
			return
		}

		err = s.setupGreetMsgHandler()
		if err != nil {
			return
		}

		err = s.setupAddressMessageHandler()
		if err != nil {
			return
		}
	})

	if err != nil {
		s.Close()
	}

	return err
}

func (s *Subscriber) sendStartMessage() error {
	msg := &nats.Msg{
		Subject: "service-discovery.start",
		Data:    s.mapSubOpts(),
	}

	err := s.natsClient.PublishMsg(msg)
	if err != nil {
		return errors.Wrap(err, "unable to publish a start message")
	}

	return nil
}

func (s *Subscriber) Close() {
	if s.natsClient != nil {
		s.natsClient.Close()
	}
}

func (s *Subscriber) setupGreetMsgHandler() error {
	discoveryMessageJson := s.mapSubOpts()

	_, err := s.natsClient.Subscribe("service-discovery.greet", nats.MsgHandler(func(greetMsg *nats.Msg) {
		msg := &nats.Msg{
			Subject: greetMsg.Reply,
			Data:    discoveryMessageJson,
		}

		_ = s.natsClient.PublishMsg(msg)
	}))

	if err != nil {
		s.logger.Error("setupGreetMsgHandler unable to subscribe to greet messages", err)
		return err
	}

	err = s.natsClient.Flush()
	if err != nil {
		s.logger.Error("setupGreetMsgHandler unable to flush subscribe greet message", err)
		return err
	}

	return nil
}

func (s *Subscriber) setupAddressMessageHandler() error {
	_, err := s.natsClient.Subscribe("service-discovery.register", nats.MsgHandler(func(msg *nats.Msg) {
		registryMessage := &RegistryMessage{}
		err := json.Unmarshal(msg.Data, registryMessage)
		if err != nil || registryMessage.IP == "" || len(registryMessage.InfraNames) == 0 {
			s.logger.Info("setupAddressMessageHandler received a malformed message", lager.Data(map[string]interface{}{
				"msgJson": string(msg.Data),
			}))
			return
		}
		s.table.Add(registryMessage.InfraNames, registryMessage.IP)
	}))

	if err != nil {
		s.logger.Error("setupAddressMessageHandler unable to subscribe to service-discovery.register", err)
		return err
	}

	_, err = s.natsClient.Subscribe("service-discovery.unregister", nats.MsgHandler(func(msg *nats.Msg) {
		registryMessage := &RegistryMessage{}
		err := json.Unmarshal(msg.Data, registryMessage)
		if err != nil || len(registryMessage.InfraNames) == 0 {
			s.logger.Info("setupAddressMessageHandler received a malformed message", lager.Data(map[string]interface{}{
				"msgJson": string(msg.Data),
			}))
			return
		}
		s.table.Remove(registryMessage.InfraNames, registryMessage.IP)
	}))

	if err != nil {
		s.logger.Error("setupAddressMessageHandler unable to subscribe to service-discovery.unregister", err)
		return err
	}

	return nil
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
