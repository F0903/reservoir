package config

import (
	"errors"
	"log/slog"
	"reservoir/config/flags"
	"reservoir/utils"
	"reservoir/utils/assertedpath"
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
	fl.AddString("listen", ":9999", "The address and port that the proxy will listen on").OnSet(func(val flags.FlagValue) { Global.ProxyListen.Overwrite(val.AsString()) })
	fl.AddString("ca-cert", "ssl/ca.crt", "Path to CA certificate file").OnSet(func(val flags.FlagValue) { Global.CaCert.Overwrite(val.AsString()) })
	fl.AddString("ca-key", "ssl/ca.key", "Path to CA private key file").OnSet(func(val flags.FlagValue) { Global.CaKey.Overwrite(val.AsString()) })
	fl.AddString("cache-dir", "var/cache/", "Path to cache directory").OnSet(func(val flags.FlagValue) { Global.CacheDir.Overwrite(val.AsString()) })
	fl.AddString("webserver-listen", "localhost:8080", "The address and port that the webserver (dashboard and API) will listen on").OnSet(func(val flags.FlagValue) { Global.WebserverListen.Overwrite(val.AsString()) })
	fl.AddBool("no-dashboard", false, "Disable the dashboard. The API must also be enabled if the dashboard is enabled.").OnSet(func(val flags.FlagValue) { Global.DashboardDisabled.Overwrite(val.AsBool()) })
	fl.AddBool("no-api", false, "Disable the API").OnSet(func(val flags.FlagValue) { Global.ApiDisabled.Overwrite(val.AsBool()) })
	fl.AddString("log-level", "", "Set the logging level (DEBUG, INFO, WARN, ERROR)").OnSet(func(val flags.FlagValue) {
		level := utils.StringToLogLevel(val.AsString())
		Global.LogLevel.Overwrite(level)
	})
	fl.AddString("log-file", "", "Set the log file path").OnSet(func(val flags.FlagValue) { Global.LogFile.Overwrite(val.AsString()) })
	fl.AddString("log-file-max-size", "", "Set the log file max size").OnSet(func(val flags.FlagValue) { Global.LogFileMaxSize.Overwrite(val.AsBytesize()) })
	fl.AddInt("log-file-max-backups", 3, "Set the log file max backups").OnSet(func(val flags.FlagValue) { Global.LogFileMaxBackups.Overwrite(val.AsInt()) })
	fl.AddBool("log-file-compress", true, "Set the log file compress").OnSet(func(val flags.FlagValue) { Global.LogFileCompress.Overwrite(val.AsBool()) })
	fl.AddBool("log-to-stdout", false, "Enable logging to stdout").OnSet(func(val flags.FlagValue) { Global.LogToStdout.Overwrite(val.AsBool()) })
	fl.Parse()
}
