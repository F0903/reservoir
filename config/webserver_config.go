package config

import (
	"fmt"
	"net/netip"
	"strings"
)

type WebserverConfig struct {
	Listen            ConfigProp[string] `json:"listen"`             // The address and port that the webserver (dashboard and API) will listen on.
	DashboardDisabled ConfigProp[bool]   `json:"dashboard_disabled"` // If true, the dashboard will be disabled.
	ApiDisabled       ConfigProp[bool]   `json:"api_disabled"`       // If true, the API will be disabled.
	TrustedProxies    ConfigProp[string] `json:"trusted_proxies"`    // Comma-separated IPs or CIDR ranges of reverse proxies in front of the webserver. X-Forwarded-For is only believed from these.
}

func parseTrustedProxies(raw string) ([]netip.Prefix, error) {
	var prefixes []netip.Prefix
	for entry := range strings.SplitSeq(raw, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}

		if addr, err := netip.ParseAddr(entry); err == nil {
			addr = addr.Unmap()
			prefixes = append(prefixes, netip.PrefixFrom(addr, addr.BitLen()))
			continue
		}

		prefix, err := netip.ParsePrefix(entry)
		if err != nil {
			return nil, fmt.Errorf("webserver.trusted_proxies has an invalid IP or CIDR range %q", entry)
		}
		prefixes = append(prefixes, prefix)
	}
	return prefixes, nil
}

// TrustedProxyPrefixes returns the configured proxies. An invalid value trusts nobody.
func (c *WebserverConfig) TrustedProxyPrefixes() []netip.Prefix {
	prefixes, _ := parseTrustedProxies(c.TrustedProxies.Read())
	return prefixes
}

func (c *WebserverConfig) setRestartNeededProps() {
	c.Listen.SetRequiresRestart()
	c.DashboardDisabled.SetRequiresRestart()
	c.ApiDisabled.SetRequiresRestart()
}

func (c *WebserverConfig) verify() error {
	if c.Listen.Read() == "" {
		return fmt.Errorf("webserver.listen cannot be empty")
	}
	if _, err := parseTrustedProxies(c.TrustedProxies.Read()); err != nil {
		return err
	}
	return nil
}

func defaultWebserverConfig() WebserverConfig {
	return WebserverConfig{
		Listen:            NewConfigProp("localhost:8080"),
		DashboardDisabled: NewConfigProp(false),
		ApiDisabled:       NewConfigProp(false),
		TrustedProxies:    NewConfigProp(""),
	}
}
