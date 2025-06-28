package config

import (
	"apt_cacher_go/utils/asserted_path"
	"encoding/json"
	"log"
	"os"
)

// Config holds the global configuration for the proxy.

var configPath = asserted_path.Assert("var/config.json")

var Global *Config = func() *Config {
	cfg, err := LoadOrDefault(configPath.GetPath())
	if err != nil {
		log.Panicf("Failed to load global config: %v", err)
	}
	return cfg
}()

type Config struct {
	IgnoreNoCache bool
}

func Default() *Config {
	return &Config{
		IgnoreNoCache: true, // Since this is geared towards caching apt repositories, we aggressively cache responses.
	}
}

func Load(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var cfg Config
	if err := json.NewDecoder(f).Decode(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func LoadOrDefault(path string) (*Config, error) {
	cfg, err := Load(path)
	if err != nil {
		log.Printf("Failed to load config from '%s': %v. Using default configuration.", path, err)
		cfg = Default()
	}
	return cfg, nil
}
