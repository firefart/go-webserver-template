package main

import (
	"fmt"
	"time"

	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/structs"
	"github.com/knadh/koanf/v2"
)

type Configuration struct {
	Server     ConfigServer  `koanf:"server"`
	Cache      ConfigCache   `koanf:"cache"`
	Timeout    time.Duration `koanf:"timeout"`
	Cloudflare bool          `koanf:"cloudflare"`
}

type ConfigServer struct {
	Port            int           `koanf:"port"`
	GracefulTimeout time.Duration `koanf:"graceful_timeout"`
}

type ConfigCache struct {
	Enabled bool          `koanf:"enabled"`
	Timeout time.Duration `koanf:"timeout"`
}

var defaultConfig = Configuration{
	Server: ConfigServer{
		Port:            8000,
		GracefulTimeout: 10 * time.Second,
	},
	Cache: ConfigCache{
		Enabled: true,
		Timeout: 1 * time.Hour,
	},
	Timeout:    5 * time.Second,
	Cloudflare: false,
}

func GetConfig(f string) (Configuration, error) {
	var k = koanf.NewWithConf(koanf.Conf{
		Delim: ".",
	})

	if err := k.Load(structs.Provider(defaultConfig, "koanf"), nil); err != nil {
		return Configuration{}, err
	}

	if err := k.Load(file.Provider(f), json.Parser()); err != nil {
		return Configuration{}, err
	}

	var config Configuration
	if err := k.Unmarshal("", &config); err != nil {
		return Configuration{}, err
	}

	// check some stuff
	if config.Server.Port == 0 {
		return Configuration{}, fmt.Errorf("please supply a port to listen on")
	}

	return config, nil
}
