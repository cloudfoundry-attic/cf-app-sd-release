package config

import (
	"encoding/json"
	"net/url"
	"fmt"
)

type Config struct {
	Address string `json:"address"`
	Port    string `json:"port"`
	Nats    []NatsConfig `json:"nats"`
}

type NatsConfig struct {
	Host string `json:"host"`
	Port uint16 `json:"port"`
	User string `json:"user"`
	Pass string `json:"pass"`
}

func NewConfig(configJSON []byte) (*Config, error) {
	sdcConfig := &Config{}
	err := json.Unmarshal(configJSON, sdcConfig)
	return sdcConfig, err
}

func (c *Config) NatsServers() []string {
	var natsServers []string
	for _, info := range c.Nats {
		uri :=
			url.URL{
				Scheme: "nats",
				User:   url.UserPassword(info.User, info.Pass),
				Host:   fmt.Sprintf("%s:%d", info.Host, info.Port),
			}
		natsServers = append(natsServers, uri.String())

	}

	return natsServers
}
