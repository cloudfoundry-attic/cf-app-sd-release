package config

import "encoding/json"

type Config struct {
	Address                           string `json:"address"`
	Port                              string `json:"port"`
	ServiceDiscoveryControllerAddress string `json:"service_discovery_controller_address"`
	ServiceDiscoveryControllerPort    string `json:"service_discovery_controller_port"`
}

func NewConfig(configJSON []byte) (*Config, error) {
	adapterConfig := &Config{}
	err := json.Unmarshal(configJSON, adapterConfig)
	return adapterConfig, err
}
