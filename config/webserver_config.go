package config

import "fmt"

type WebserverConfig struct {
	Listen            ConfigProp[string] `json:"listen"`             // The address and port that the webserver (dashboard and API) will listen on.
	DashboardDisabled ConfigProp[bool]   `json:"dashboard_disabled"` // If true, the dashboard will be disabled.
	ApiDisabled       ConfigProp[bool]   `json:"api_disabled"`       // If true, the API will be disabled.
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
	return nil
}

func defaultWebserverConfig() WebserverConfig {
	return WebserverConfig{
		Listen:            NewConfigProp("localhost:8080"),
		DashboardDisabled: NewConfigProp(false),
		ApiDisabled:       NewConfigProp(false),
	}
}
