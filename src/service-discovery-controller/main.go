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

	"code.cloudfoundry.org/lager"
	"strings"
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

	logger := lager.NewLogger("service-discovery-controller")

	addressTable := addresstable.NewAddressTable()

	subscriber := launchSubscriber(config, addressTable, logger)

	launchHttpServer(config, addressTable, logger)

	fmt.Println("Server Started")

	select {
	case <-signalChannel:
		subscriber.Close() //TODO: test?
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
				fmt.Fprintln(os.Stderr, "Error writing to http response body") // not tested
			}
		}))
	}()
}

func launchSubscriber(config *config.Config, addressTable *addresstable.AddressTable, logger lager.Logger) *mbus.Subscriber {
	subOpts := mbus.SubscriberOpts{ //TODO: does this need to be configurable? can we hard code into subscriber?
		ID: "Fake-Subscriber-ID",
		MinimumRegisterIntervalInSeconds: 60,
		PruneThresholdInSeconds:          120,
	}

	provider := &mbus.NatsConnWithUrlProvider{
		Url: strings.Join(config.NatsServers(), ","), //TODO: test me, joining multiple urls (inject a bad and good server?)
	}

	localIP, err := LocalIP()
	if err != nil {
		panic(fmt.Sprintf("failed to get local IP: %v", err)) //TODO: handle err
	}

	subscriber := mbus.NewSubscriber(provider, subOpts, addressTable, localIP, logger)

	err = subscriber.Run()
	if err != nil {
		panic(fmt.Sprintf("Subscriber: CANT RUN!: %v", err)) //TODO: panic
	}
	return subscriber
}

func LocalIP() (string, error) {
	addr, err := net.ResolveUDPAddr("udp", "1.2.3.4:1")
	if err != nil {
		return "", err
	}

	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return "", err
	}

	defer conn.Close()

	host, _, err := net.SplitHostPort(conn.LocalAddr().String())
	if err != nil {
		return "", err
	}

	return host, nil
}
