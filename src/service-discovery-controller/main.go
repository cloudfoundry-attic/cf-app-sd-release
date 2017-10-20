package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path"
	"service-discovery-controller/addresstable"
	"service-discovery-controller/config"
	"service-discovery-controller/mbus"
	"syscall"

	"service-discovery-controller/localip"
	"strings"

	"code.cloudfoundry.org/cf-networking-helpers/middleware/adapter"
	"code.cloudfoundry.org/lager"
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

func main() {
	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, syscall.SIGTERM, os.Interrupt)
	configPath := flag.String("c", "", "path to config file")
	flag.Parse()

	logger := lager.NewLogger("service-discovery-controller")
	logger.RegisterSink(lager.NewWriterSink(os.Stdout, lager.DEBUG))

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

	addressTable := addresstable.NewAddressTable()

	subscriber, err := launchSubscriber(config, addressTable, logger)
	if err != nil {
		logger.Error("Failed to launch subscriber", err)
		os.Exit(2)
	}

	launchHttpServer(config, addressTable, logger)

	fmt.Println("Server Started")

	select {
	case <-signalChannel:
		subscriber.Close()
		fmt.Println("Shutting service-discovery-controller down")
		return
	}
}
func launchHttpServer(config *config.Config, addressTable *addresstable.AddressTable, logger lager.Logger) {
	address := fmt.Sprintf("%s:%s", config.Address, config.Port)
	l, err := net.Listen("tcp", address)
	if err != nil {
		fmt.Fprintln(os.Stderr, fmt.Sprintf("Address (%s) not available", address))
		os.Exit(1)
	}

	go func() {
		http.Serve(l, http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
			serviceKey := path.Base(req.URL.Path)

			ips := addressTable.Lookup(serviceKey)
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
				logger.Debug("Error writing to http response body")
			}

			logger.Debug("HTTPServer access", lager.Data(map[string]interface{}{
				"serviceKey":   serviceKey,
				"responseJson": string(json),
			}))
		}))
	}()
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

	subscriber := mbus.NewSubscriber(provider, subOpts, addressTable, localIP, logger)

	err = subscriber.Run()
	if err != nil {
		return subscriber, err
	}

	return subscriber, nil
}
