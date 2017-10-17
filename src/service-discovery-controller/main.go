package main

import (
	"encoding/json"
	"strings"

	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path"
	"service-discovery-controller/addresstable"
	"service-discovery-controller/config"
	"service-discovery-controller/mbus"
	"sync/atomic"
	"syscall"
	"time"

	"code.cloudfoundry.org/lager"
	"github.com/nats-io/nats"
)

type host struct {
	IPAddress       string                 `json:"ip_address"`
	LastCheckIn     string                 `json:"last_check_in"`
	Port            int32                  `json:"port"`
	Revision        string                 `json:"revision"`
	Service         string                 `json:"service"`
	ServiceRepoName string                 `json:"service_repo_name"`
	Tags            map[string]interface{} `json:"tags"`
}

type registration struct {
	Hosts   []host `json:"hosts"`
	Env     string `json:"env"`
	Service string `json:"service"`
}

var (
	jobPrefix = "service-discovery-controller"
	logPrefix = "service-discovery"
)

func main() {
	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, syscall.SIGTERM, os.Interrupt)
	configPath := flag.String("c", "", "path to config file")
	flag.Parse()

	bytes, err := ioutil.ReadFile(*configPath)
	if err != nil {
		fmt.Printf("Could not read config file at path '%s'", *configPath)
		os.Exit(2)
	}

	config, err := config.NewConfig(bytes)
	if err != nil {
		fmt.Printf("Could not parse config file at path '%s'", *configPath)
		os.Exit(2)
	}

	logger := lager.NewLogger(fmt.Sprintf("%s.%s", logPrefix, jobPrefix))
	reconfigurableSink := initLoggerSink(logger, config.LogLevel)
	logger.RegisterSink(reconfigurableSink)

	address := fmt.Sprintf("%s:%s", config.Address, config.Port)
	l, err := net.Listen("tcp", address)
	if err != nil {
		fmt.Fprintln(os.Stderr, fmt.Sprintf("Address (%s) not available", address))
		os.Exit(1)
	}

	subOpts := mbus.SubscriberOpts{
		ID: "Fake-Subscriber-ID",
		MinimumRegisterIntervalInSeconds: 60,
		PruneThresholdInSeconds:          120,
	}

	startMsgChan := make(chan struct{})
	natsClient := connectToNatsServer(logger.Session("nats"), config, startMsgChan)

	addressTable := addresstable.NewAddressTable()

	subscriber := mbus.NewSubscriber(natsClient, subOpts, addressTable, logger.Session("subscriber"))
	natsServers := config.NatsServers()

	routes := map[string][]string{
		"app-id.internal.local.": {
			"192.168.0.1",
			"192.168.0.2",
		},
		"large-id.internal.local.": {
			"192.168.0.1",
			"192.168.0.2",
			"192.168.0.3",
			"192.168.0.4",
			"192.168.0.5",
			"192.168.0.6",
			"192.168.0.7",
			"192.168.0.8",
			"192.168.0.9",
			"192.168.0.10",
			"192.168.0.11",
			"192.168.0.12",
			"192.168.0.13",
		},
	}

	go func() {
		http.Serve(l, http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
			serviceKey := path.Base(req.URL.Path)

			ips := routes[serviceKey]
			hosts := []host{}
			for _, ip := range ips {
				hosts = append(hosts, host{
					IPAddress: ip,
					Tags:      make(map[string]interface{}),
				})
			}

			var err error
			json, err := json.Marshal(registration{Hosts: hosts})
			if err != nil {
				http.Error(resp, err.Error(), http.StatusInternalServerError)
				return
			}

			_, err = resp.Write(json)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error writing to http response body") // not tested
			}
		}))
	}()

	fmt.Println("Server Started")
	subscriber.SendStartMessage(natsServers[0])

	select {
	case <-signalChannel:
		fmt.Println("Shutting service-discovery-controller down")
		return
	}
}

func natsOptions(logger lager.Logger, c *config.Config, natsHost *atomic.Value, startMsg chan<- struct{}) nats.Options {
	natsServers := c.NatsServers()

	options := nats.DefaultOptions
	options.Servers = natsServers
	options.PingInterval = 20 * time.Second
	options.MaxReconnect = -1
	connectedChan := make(chan struct{})

	options.ClosedCB = func(conn *nats.Conn) {
		logger.Fatal(
			"nats-connection-closed",
			errors.New("unexpected close"),
			lager.Data{"last_error": conn.LastError()},
		)
	}

	options.DisconnectedCB = func(conn *nats.Conn) {
		hostStr := natsHost.Load().(string)
		logger.Info("nats-connection-disconnected", lager.Data{"nats-host": hostStr})

		go func() {
			ticker := time.NewTicker(20 * time.Second)

			for {
				select {
				case <-connectedChan:
					return
				case <-ticker.C:
					logger.Info("nats-connection-still-disconnected")
				}
			}
		}()
	}

	options.ReconnectedCB = func(conn *nats.Conn) {
		connectedChan <- struct{}{}

		natsURL, err := url.Parse(conn.ConnectedUrl())
		natsHostStr := ""
		if err != nil {
			logger.Error("nats-url-parse-error", err)
		} else {
			natsHostStr = natsURL.Host
		}
		natsHost.Store(natsHostStr)

		logger.Info("nats-connection-reconnected", lager.Data{"nats-host": natsHostStr})
		startMsg <- struct{}{}
	}

	return options
}

func connectToNatsServer(logger lager.Logger, c *config.Config, startMsg chan<- struct{}) *nats.Conn {
	var natsClient *nats.Conn
	var natsHost atomic.Value
	var err error

	options := natsOptions(logger, c, &natsHost, startMsg)
	attempts := 3
	for attempts > 0 {
		natsClient, err = options.Connect()
		if err == nil {
			break
		} else {
			attempts--
			time.Sleep(100 * time.Millisecond)
		}
	}

	if err != nil {
		logger.Fatal("nats-connection-error", err)
	}

	var natsHostStr string
	natsURL, err := url.Parse(natsClient.ConnectedUrl())
	if err == nil {
		natsHostStr = natsURL.Host
	}

	logger.Info("Successfully-connected-to-nats", lager.Data{"host": natsHostStr})

	natsHost.Store(natsHostStr)
	return natsClient
}

const (
	DEBUG = "debug"
	INFO  = "info"
	ERROR = "error"
	FATAL = "fatal"
)

func initLoggerSink(logger lager.Logger, level string) *lager.ReconfigurableSink {
	var logLevel lager.LogLevel
	switch strings.ToLower(level) {
	case DEBUG:
		logLevel = lager.DEBUG
	case INFO:
		logLevel = lager.INFO
	case ERROR:
		logLevel = lager.ERROR
	case FATAL:
		logLevel = lager.FATAL
	default:
		logLevel = lager.INFO
	}
	w := lager.NewWriterSink(os.Stdout, lager.DEBUG)
	return lager.NewReconfigurableSink(w, logLevel)
}
