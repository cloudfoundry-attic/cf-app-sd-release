package sdcclient

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
)

type ServiceDiscoveryClient struct {
	serverURL string
}

type serverResponse struct {
	Hosts []host `json:"Hosts"`
}

type host struct {
	IPAddress string `json:"ip_address"`
}

func NewServiceDiscoveryClient(serverURL string) *ServiceDiscoveryClient {
	return &ServiceDiscoveryClient{
		serverURL: serverURL,
	}
}

func (s *ServiceDiscoveryClient) IPs(infrastructureName string) ([]string, error) {
	requestUrl := fmt.Sprintf("%s/v1/registration/%s", s.serverURL, infrastructureName)

	httpResp, err := http.Get(requestUrl)
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
