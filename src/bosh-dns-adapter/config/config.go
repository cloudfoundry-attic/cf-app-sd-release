package config

import "encoding/json"

type Config struct {
	Address string
	Port    string
}

func NewConfig(configJSON []byte) (*Config, error) {
	adapterConfig := &Config{}
	err := json.Unmarshal(configJSON, adapterConfig)
	return adapterConfig, err
}
