package config

import "encoding/json"

type Config struct {
	Address                           string `json:"address"`
	Port                              string `json:"port"`
}

func NewConfig(configJSON []byte) (*Config, error) {
	sdcConfig := &Config{}
	err := json.Unmarshal(configJSON, sdcConfig)
	return sdcConfig, err
}
