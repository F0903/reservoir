package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"reservoir/config/flags"
	"reservoir/utils"
	"reservoir/utils/assertedpath"
	"reservoir/version"
)

var (
	ErrUpdateFailed = errors.New("failed to update global config")
)

var configPath = assertedpath.Assert("var/config.json")

// ! NEVER COPY THIS!
var Global *Config = func() *Config {
	cfg, err := loadOrDefault(configPath.Path)
	if err != nil {
		slog.Error("Failed to load global config", "error", err)
		panic(err)
	}
	return cfg
}()

func OverrideGlobalConfigFromFlags() {
	fl := flags.New()

	fl.AddBool("version", false, "Print the version and exit").OnSet(func(val flags.FlagValue) {
		if val.AsBool() {
			fmt.Printf("Reservoir version: %s\n", version.Version)
			os.Exit(0)
		}
	})

	fl.AddString("listen", ":9999", "The address and port that the proxy will listen on").OnSet(func(val flags.FlagValue) {
		Global.Proxy.Listen.Overwrite(val.AsString())
	})
	fl.AddString("ca-cert", "ssl/ca.crt", "Path to CA certificate file").OnSet(func(val flags.FlagValue) {
		Global.Proxy.CaCert.Overwrite(val.AsString())
	})
	fl.AddString("ca-key", "ssl/ca.key", "Path to CA private key file").OnSet(func(val flags.FlagValue) {
		Global.Proxy.CaKey.Overwrite(val.AsString())
	})

	fl.AddString("cache-dir", "var/cache/", "Path to cache directory").OnSet(func(val flags.FlagValue) {
		Global.Cache.File.Dir.Overwrite(val.AsString())
	})

	fl.AddString("webserver-listen", "localhost:8080", "The address and port that the webserver (dashboard and API) will listen on").OnSet(func(val flags.FlagValue) {
		Global.Webserver.Listen.Overwrite(val.AsString())
	})
	fl.AddBool("no-dashboard", false, "Disable the dashboard. The API must also be enabled if the dashboard is enabled.").OnSet(func(val flags.FlagValue) {
		Global.Webserver.DashboardDisabled.Overwrite(val.AsBool())
	})
	fl.AddBool("no-api", false, "Disable the API").OnSet(func(val flags.FlagValue) {
		Global.Webserver.ApiDisabled.Overwrite(val.AsBool())
	})

	fl.AddString("log-level", "", "Set the logging level (DEBUG, INFO, WARN, ERROR)").OnSet(func(val flags.FlagValue) {
		level := utils.StringToLogLevel(val.AsString())
		Global.Logging.Level.Overwrite(level)
	})
	fl.AddString("log-file", "", "Set the log file path").OnSet(func(val flags.FlagValue) {
		Global.Logging.File.Overwrite(val.AsString())
	})
	fl.AddString("log-file-max-size", "", "Set the log file max size").OnSet(func(val flags.FlagValue) {
		Global.Logging.MaxSize.Overwrite(val.AsBytesize())
	})
	fl.AddInt("log-file-max-backups", 3, "Set the log file max backups").OnSet(func(val flags.FlagValue) {
		Global.Logging.MaxBackups.Overwrite(val.AsInt())
	})
	fl.AddBool("log-file-compress", true, "Set the log file compress").OnSet(func(val flags.FlagValue) {
		Global.Logging.Compress.Overwrite(val.AsBool())
	})
	fl.AddBool("log-to-stdout", false, "Enable logging to stdout").OnSet(func(val flags.FlagValue) {
		Global.Logging.ToStdout.Overwrite(val.AsBool())
	})

	fl.Parse()
}
