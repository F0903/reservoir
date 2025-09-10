package config

import (
	"log/slog"
	"sync/atomic"
)

var restartNeeded atomic.Bool

func setRestartNeeded() {
	if IsRestartNeeded() {
		return
	}

	slog.Debug("Setting restartNeeded to true")
	restartNeeded.Store(true)
}

func IsRestartNeeded() bool {
	return restartNeeded.Load()
}
