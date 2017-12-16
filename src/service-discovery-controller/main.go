package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"service-discovery-controller/addresstable"
	"service-discovery-controller/config"
	"service-discovery-controller/mbus"
	"syscall"
	"time"

	"service-discovery-controller/localip"
	"strings"

	"code.cloudfoundry.org/cf-networking-helpers/lagerlevel"
	"code.cloudfoundry.org/cf-networking-helpers/metrics"
	"code.cloudfoundry.org/cf-networking-helpers/middleware/adapter"
	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager"
	"github.com/cloudfoundry/dropsonde"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/grouper"
	"github.com/tedsuo/ifrit/sigmon"
	"service-discovery-controller/routes"
)

func main() {
	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, syscall.SIGTERM, os.Interrupt)
	configPath := flag.String("c", "", "path to config file")
	flag.Parse()

	logger := lager.NewLogger("service-discovery-controller")
	writerSink := lager.NewWriterSink(os.Stdout, lager.DEBUG)
	sink := lager.NewReconfigurableSink(writerSink, lager.INFO)
	logger.RegisterSink(sink)

	var err error
	bytes, err := ioutil.ReadFile(*configPath)
	if err != nil {
		logger.Error(fmt.Sprintf("Could not read config file at path '%s'", *configPath), err)
		os.Exit(2)
	}

	config, err := config.NewConfig(bytes)
	if err != nil {
		logger.Error(fmt.Sprintf("Could not parse config file at path '%s'", *configPath), err)
		os.Exit(2)
	}

	addressTable := addresstable.NewAddressTable(
		time.Duration(config.StalenessThresholdSeconds)*time.Second,
		time.Duration(config.PruningIntervalSeconds)*time.Second,
		clock.NewClock(),
		logger.Session("address-table"))

	metronAddress := fmt.Sprintf("127.0.0.1:%d", config.MetronPort)
	err = dropsonde.Initialize(metronAddress, "service-discovery-controller")
	if err != nil {
		panic(err)
	}

	subscriber, err := launchSubscriber(config, addressTable, logger)
	if err != nil {
		logger.Error("Failed to launch subscriber", err)
		os.Exit(2)
	}

	uptimeSource := metrics.NewUptimeSource()
	metricsEmitter := metrics.NewMetricsEmitter(
		logger,
		time.Duration(config.MetricsEmitSeconds)*time.Second,
		uptimeSource,
	)

	members := grouper.Members{
		{"metrics-emitter", metricsEmitter},
		{"log-level-server", lagerlevel.NewServer(config.LogLevelAddress, config.LogLevelPort, sink, logger.Session("log-level-server"))},
		{"routes-server", routes.NewServer(addressTable, config, logger.Session("routes-server"))},
	}
	group := grouper.NewOrdered(os.Interrupt, members)
	monitor := ifrit.Invoke(sigmon.New(group))

	go func() {
		err = <-monitor.Wait()
		if err != nil {
			logger.Fatal("ifrit-failure", err)
		}
	}()

	logger.Info("server-started")

	select {
	case signal := <-signalChannel:
		subscriber.Close()
		addressTable.Shutdown()
		monitor.Signal(signal)
		fmt.Println("Shutting service-discovery-controller down")
		return
	}
}

func launchSubscriber(config *config.Config, addressTable *addresstable.AddressTable, logger lager.Logger) (*mbus.Subscriber, error) {
	uuidGenerator := adapter.UUIDAdapter{}

	uuid, err := uuidGenerator.GenerateUUID()
	if err != nil {
		return &mbus.Subscriber{}, err
	}

	subscriberID := fmt.Sprintf("%s-%s", config.Index, uuid)

	subOpts := mbus.SubscriberOpts{
		ID: subscriberID,
		MinimumRegisterIntervalInSeconds: 60,
		PruneThresholdInSeconds:          120,
	}

	provider := &mbus.NatsConnWithUrlProvider{
		Url: strings.Join(config.NatsServers(), ","),
	}

	localIP, err := localip.LocalIP()
	if err != nil {
		return &mbus.Subscriber{}, err
	}

	metricsSender := &metrics.MetricsSender{
		Logger: logger.Session("metrics"),
	}

	subscriber := mbus.NewSubscriber(provider, subOpts, addressTable, localIP, logger.Session("mbus"), metricsSender)

	err = subscriber.Run()
	if err != nil {
		return subscriber, err
	}

	return subscriber, nil
}
