package config

import (
	"log/slog"
	"reservoir/utils/bytesize"
)

type LogConfig struct {
	Level      ConfigProp[slog.Level]        `json:"level"`       // The log level to use for the application.
	File       ConfigProp[string]            `json:"file"`        // The path to the log file.
	MaxSize    ConfigProp[bytesize.ByteSize] `json:"max_size"`    // The maximum size of the log file.
	MaxBackups ConfigProp[int]               `json:"max_backups"` // The maximum number of old log files to retain.
	Compress   ConfigProp[bool]              `json:"compress"`    // If true, old log files will be compressed.
	ToStdout   ConfigProp[bool]              `json:"to_stdout"`   // If true, log messages will be written to stdout.
}

func (c *LogConfig) setRestartNeededProps() {
	// Everything in logging is currently hot-swappable!
}

func defaultLogConfig() LogConfig {
	return LogConfig{
		Level:      NewConfigProp(slog.LevelInfo),
		File:       NewConfigProp("var/proxy.log"),
		MaxSize:    NewConfigProp(bytesize.ParseUnchecked("500M")),
		MaxBackups: NewConfigProp(3),
		Compress:   NewConfigProp(true),
		ToStdout:   NewConfigProp(false),
	}
}
