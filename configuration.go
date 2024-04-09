package main

import (
	"github.com/BurntSushi/toml"
)

type Config struct {
	Address string `toml:"address"`
	Port    int    `toml:"port"`
}

func loadConfig(path string) (config *Config, err error) {
	config = &Config{}
	_, err = toml.DecodeFile(path, config)
	if err != nil {
		return nil, err
	}

	if config.Port == 0 {
		config.Port = 8080
	}

	if config.Address == "" {
		config.Address = "0.0.0.0"
	}

	return config, nil
}
