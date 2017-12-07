package main

import (
	"bosh-dns-adapter/config"
	"bosh-dns-adapter/sdcclient"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/net/dns/dnsmessage"
	"github.com/cloudfoundry/dropsonde"
	"time"
	"github.com/tedsuo/ifrit/grouper"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/sigmon"
	"code.cloudfoundry.org/cf-networking-helpers/metrics"
	"code.cloudfoundry.org/lager"
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

	address := fmt.Sprintf("%s:%s", config.Address, config.Port)
	l, err := net.Listen("tcp", address)
	if err != nil {
		fmt.Fprintln(os.Stderr, fmt.Sprintf("Address (%s) not available", address))
		os.Exit(1)
	}

	sdcServerUrl := fmt.Sprintf("https://%s:%s",
		config.ServiceDiscoveryControllerAddress,
		config.ServiceDiscoveryControllerPort,
	)

	metronAddress := fmt.Sprintf("127.0.0.1:%d", config.MetronPort)
	err = dropsonde.Initialize(metronAddress, "bosh-dns-adapter")
	if err != nil {
		panic(err)
	}

	sdcClient, err := sdcclient.NewServiceDiscoveryClient(sdcServerUrl, config.CACert, config.ClientCert, config.ClientKey)
	if err != nil {
		fmt.Fprintln(os.Stderr, fmt.Sprintf("Unable to create service discovery client: %s", err))
		os.Exit(1)
	}

	go func() {
		http.Serve(l, http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
			dnsType := getQueryParam(req, "type", "1")
			name := getQueryParam(req, "name", "")

			if dnsType != "1" {
				writeResponse(resp, dnsmessage.RCodeSuccess, name, dnsType, nil)
				return
			}

			if name == "" {
				resp.WriteHeader(http.StatusBadRequest)
				writeResponse(resp, dnsmessage.RCodeServerFailure, name, dnsType, nil)
				return
			}

			ips, err := sdcClient.IPs(name)
			if err != nil {
				writeErrorResponse(resp, errors.New(fmt.Sprintf("Error querying Service Discover Controller: %s", err)))
				return
			}

			writeResponse(resp, dnsmessage.RCodeSuccess, name, dnsType, ips)
		}))
	}()

	uptimeSource := metrics.NewUptimeSource()
	metricsEmitter := metrics.NewMetricsEmitter(
		lager.NewLogger("bosh-dns-adapter"),
		time.Duration(config.MetricsEmitSeconds)*time.Second,
		uptimeSource,
	)
	members := grouper.Members{
		{"metrics-emitter", metricsEmitter},
	}
	group := grouper.NewOrdered(os.Interrupt, members)
	monitor := ifrit.Invoke(sigmon.New(group))

	go func() {
		err = <-monitor.Wait()
		if err != nil {
			fmt.Fprintln(os.Stderr, fmt.Sprintf("ifrit-failure: %s", err))
			os.Exit(1)
		}
	}()

	fmt.Println("Server Started")
	select {
	case <-signalChannel:
		fmt.Println("Shutting bosh-dns-adapter down")
		return
	}
}
func getQueryParam(req *http.Request, key, defaultValue string) string {
	queryValue := req.URL.Query().Get(key)
	if queryValue == "" {
		return defaultValue
	}

	return queryValue
}

type ServiceDiscoveryClient interface {
	IPs(infraName string) ([]string, error)
}

func writeErrorResponse(resp http.ResponseWriter, err error) {
	resp.WriteHeader(http.StatusInternalServerError)
	_, err = resp.Write([]byte(err.Error()))
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error writing to http response body") // not tested
	}
}

func writeResponse(resp http.ResponseWriter, dnsResponseStatus dnsmessage.RCode, requestedInfraName string, dnsType string, ips []string) {
	responseBody, err := buildResponseBody(dnsResponseStatus, requestedInfraName, dnsType, ips)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error building response: %v", err) // not tested
		return
	}

	_, err = resp.Write([]byte(responseBody))
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error writing to http response body") // not tested
	}
}

type Answer struct {
	Name   string `json:"name"`
	RRType uint16 `json:"type"`
	TTL    uint32 `json:"TTL"`
	Data   string `json:"data"`
}

func buildResponseBody(dnsResponseStatus dnsmessage.RCode, requestedInfraName string, dnsType string, ips []string) (string, error) {
	answers := make([]Answer, len(ips), len(ips))
	for i, ip := range ips {
		answers[i] = Answer{
			Name:   requestedInfraName,
			RRType: uint16(dnsmessage.TypeA),
			Data:   ip,
			TTL:    0,
		}
	}

	bytes, err := json.Marshal(answers)
	if err != nil {
		return "", err // not tested
	}

	template := `{
		"Status": %d,
		"TC": false,
		"RD": false,
		"RA": false,
		"AD": false,
		"CD": false,
		"Question":
		[
			{
				"name": "%s",
				"type": %s
			}
		],
		"Answer": %s,
		"Additional": [ ],
		"edns_client_subnet": "0.0.0.0/0"
	}`

	return fmt.Sprintf(template, dnsResponseStatus, requestedInfraName, dnsType, string(bytes)), nil
}
