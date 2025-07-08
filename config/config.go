package config

import (
	"apt_cacher_go/utils/asserted_path"
	"encoding/json"
	"fmt"
	"log"
	"os"
)

//TODO: add file watcher to reload config on changes

var configPath = asserted_path.Assert("var/config.json")

var Global *Config = func() *Config {
	cfg, err := LoadOrDefault(configPath.GetPath())
	if err != nil {
		log.Panicf("Failed to load global config: %v", err)
	}
	return cfg
}()

type Config struct {
	AlwaysCache          bool
	UpstreamDefaultHttps bool
}

func Default() *Config {
	return &Config{
		AlwaysCache:          false,
		UpstreamDefaultHttps: true,
	}
}

// Writes the configuration to disk.
func (c *Config) Persist() error {
	f, err := os.Create(configPath.GetPath())
	if err != nil {
		return fmt.Errorf("failed to open config file for writing: %v", err)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ") // Pretty print the JSON output
	if err := enc.Encode(c); err != nil {
		return fmt.Errorf("failed to write config to file: %v", err)
	}
	return nil
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
		log.Printf("Failed to load config from '%s': '%v' Using default configuration...", path, err)
		cfg = Default()
		if err := cfg.Persist(); err != nil {
			return nil, fmt.Errorf("failed to persist default config: %v", err)
		}
	}
	return cfg, nil
}
