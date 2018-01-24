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

	"service-discovery-controller/routes"

	"code.cloudfoundry.org/cf-networking-helpers/lagerlevel"
	"code.cloudfoundry.org/cf-networking-helpers/metrics"
	"code.cloudfoundry.org/cf-networking-helpers/middleware/adapter"
	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager"
	"github.com/cloudfoundry/dropsonde"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/grouper"
	"github.com/tedsuo/ifrit/sigmon"
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
		time.Duration(config.ResumePruningDelaySeconds)*time.Second,
		clock.NewClock(),
		logger.Session("address-table"))

	metronAddress := fmt.Sprintf("127.0.0.1:%d", config.MetronPort)
	err = dropsonde.Initialize(metronAddress, "service-discovery-controller")
	if err != nil {
		panic(err)
	}

	subscriber, err := buildSubscriber(config, addressTable, logger)
	if err != nil {
		logger.Error("Failed to build subscriber", err)
		os.Exit(2)
	}

	dnsRequestRecorder := &routes.MetricsRecorder{}

	dnsRequestSource := metrics.MetricSource{
		Name:   "dnsRequest",
		Unit:   "request",
		Getter: dnsRequestRecorder.Getter,
	}

	uptimeSource := metrics.NewUptimeSource()
	metricsEmitter := metrics.NewMetricsEmitter(
		logger,
		time.Duration(config.MetricsEmitSeconds)*time.Second,
		uptimeSource,
		dnsRequestSource,
	)

	metricsSender := &metrics.MetricsSender{
		Logger: logger.Session("time-metric-emitter"),
	}

	logLevelServer := lagerlevel.NewServer(
		config.LogLevelAddress,
		config.LogLevelPort,
		sink,
		logger.Session("log-level-server"),
	)

	routesServer := routes.NewServer(
		addressTable,
		config,
		dnsRequestRecorder,
		metricsSender,
		logger.Session("routes-server"),
	)

	members := grouper.Members{
		{"subscriber", subscriber},
		{"metrics-emitter", metricsEmitter},
		{"log-level-server", logLevelServer},
		{"routes-server", routesServer},
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
		logger.Info("server-stopped")
		return
	}
}

func buildSubscriber(config *config.Config, addressTable *addresstable.AddressTable, logger lager.Logger) (*mbus.Subscriber, error) {
	uuidGenerator := adapter.UUIDAdapter{}

	uuid, err := uuidGenerator.GenerateUUID()
	if err != nil {
		return &mbus.Subscriber{}, err
	}

	subscriberID := fmt.Sprintf("%s-%s", config.Index, uuid)

	subOpts := mbus.SubscriberOpts{
		ID: subscriberID,
		MinimumRegisterIntervalInSeconds: config.ResumePruningDelaySeconds,
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

	clock := clock.NewClock()
	warmDuration := time.Duration(config.WarmDurationSeconds) * time.Second

	subscriber := mbus.NewSubscriber(provider, subOpts, warmDuration, addressTable,
		localIP, logger.Session("mbus"), metricsSender, clock)
	return subscriber, nil
}
