package config

import (
	"fmt"
	"reservoir/utils/duration"
	"time"
)

type CachePolicyConfig struct {
	IgnoreCacheControl ConfigProp[bool]              `json:"ignore_cache_control"`  // If true, the proxy will ignore Cache-Control headers from the upstream response.
	DefaultMaxAge      ConfigProp[duration.Duration] `json:"default_max_age"`       // The default cache max age to use if the upstream response does not specify a Cache-Control or Expires header.
	ForceDefaultMaxAge ConfigProp[bool]              `json:"force_default_max_age"` // If true, always use the default cache max age.
}

type ProxyConfig struct {
	Listen               ConfigProp[string] `json:"listen"`                 // The address and port that the proxy will listen on.
	CaCert               ConfigProp[string] `json:"ca_cert"`                // Path to CA certificate file.
	CaKey                ConfigProp[string] `json:"ca_key"`                 // Path to CA private key file.
	UpstreamDefaultHttps ConfigProp[bool]   `json:"upstream_default_https"` // If true, the proxy will always send HTTPS instead of HTTP to the upstream server.
	RetryOnRange416      ConfigProp[bool]   `json:"retry_on_range_416"`     // If true, the proxy will retry a request without the Range header if the upstream responds with a 416 Range Not Satisfiable.
	RetryOnInvalidRange  ConfigProp[bool]   `json:"retry_on_invalid_range"` // If true, the proxy will retry a request without the Range header if the client sends an invalid Range header. (not recommended)
	CachePolicy          CachePolicyConfig  `json:"cache_policy"`
}

func (c *ProxyConfig) setRestartNeededProps() {
	c.Listen.SetRequiresRestart()
	c.CaCert.SetRequiresRestart()
	c.CaKey.SetRequiresRestart()
}

func (c *ProxyConfig) verify() error {
	if c.Listen.Read() == "" {
		return fmt.Errorf("proxy.listen cannot be empty")
	}
	if c.CaCert.Read() == "" {
		return fmt.Errorf("proxy.ca_cert cannot be empty")
	}
	if c.CaKey.Read() == "" {
		return fmt.Errorf("proxy.ca_key cannot be empty")
	}
	return nil
}

func defaultProxyConfig() ProxyConfig {
	return ProxyConfig{
		Listen:               NewConfigProp(":9999"),
		CaCert:               NewConfigProp("ssl/ca.crt"),
		CaKey:                NewConfigProp("ssl/ca.key"),
		UpstreamDefaultHttps: NewConfigProp(true),
		RetryOnRange416:      NewConfigProp(true),
		RetryOnInvalidRange:  NewConfigProp(false),
		CachePolicy: CachePolicyConfig{
			IgnoreCacheControl: NewConfigProp(true),
			DefaultMaxAge:      NewConfigProp(duration.Duration(1 * time.Hour)),
			ForceDefaultMaxAge: NewConfigProp(true),
		},
	}
}
