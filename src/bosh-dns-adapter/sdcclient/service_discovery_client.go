package sdcclient

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
)

type ServiceDiscoveryClient struct {
	serverURL string
	client    *http.Client
}

type serverResponse struct {
	Hosts []host `json:"Hosts"`
}

type host struct {
	IPAddress string `json:"ip_address"`
}

func NewServiceDiscoveryClient(serverURL, caPath, clientCertPath, clientKeyPath string) (*ServiceDiscoveryClient, error) {
	caPemBytes, err := ioutil.ReadFile(caPath)
	if err != nil {
		return nil, fmt.Errorf("read CA file: %s", err)
	}
	caCertPool := x509.NewCertPool()
	if caCertPool.AppendCertsFromPEM(caPemBytes) != true {
		return nil, fmt.Errorf("load CA file into cert pool")
	}

	cert, err := tls.LoadX509KeyPair(clientCertPath, clientKeyPath)
	if err != nil {
		return nil, fmt.Errorf("load client key pair: %s", err)
	}

	tlsConfig := &tls.Config{
		MinVersion:   tls.VersionTLS12,
		ClientCAs:    caCertPool,
		RootCAs:      caCertPool,
		Certificates: []tls.Certificate{cert},
	}

	tlsConfig.BuildNameToCertificate()
	tlsConfig.ServerName = "service-discovery-controller.internal"

	tr := &http.Transport{
		TLSClientConfig: tlsConfig,
	}
	client := &http.Client{Transport: tr}

	return &ServiceDiscoveryClient{
		serverURL: serverURL,
		client:    client,
	}, nil
}

func (s *ServiceDiscoveryClient) IPs(infrastructureName string) ([]string, error) {
	requestUrl := fmt.Sprintf("%s/v1/registration/%s", s.serverURL, infrastructureName)

	httpResp, err := s.client.Get(requestUrl)
	if err != nil {
		return []string{}, err
	}

	if httpResp.StatusCode != http.StatusOK {
		return []string{}, errors.New(fmt.Sprintf("Received non successful response from server: %+v", httpResp))
	}

	bytes, err := ioutil.ReadAll(httpResp.Body)
	if err != nil {
		return []string{}, err
	}

	var serverResponse *serverResponse
	err = json.Unmarshal(bytes, &serverResponse)
	if err != nil {
		return []string{}, err
	}

	len := len(serverResponse.Hosts)
	ips := make([]string, len, len)
	for i, host := range serverResponse.Hosts {
		ips[i] = host.IPAddress
	}

	return ips, nil
}
