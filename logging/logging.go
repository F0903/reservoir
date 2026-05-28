package logging

import (
	"bytes"
	"io"
	"log/slog"
	"os"
	"reservoir/config"
	"reservoir/logging/early"
	"reservoir/utils/bytesize"
	"sync"
)

var fileLog *fileLogger = nil // Current file logger instance if any
var logLevel slog.LevelVar
var loggerMu sync.Mutex

// Initializes a log file writer if file logging is configured.
func newLogFileWriter(cfg *config.Config) *fileLogger {
	slog.Info("Initializing log file writer...")

	logFilePath := cfg.Logging.File.Read()
	logFileMaxSize := int(cfg.Logging.MaxSize.Read().MegaBytes())
	logFileMaxBackups := cfg.Logging.MaxBackups.Read()
	logFileCompress := cfg.Logging.Compress.Read()

	if logFilePath == "" {
		slog.Info("Log file logging is disabled, skipping log file writer initialization")
		return nil
	}

	logFile := newFileLogger(logFilePath, logFileMaxSize, logFileMaxBackups, logFileCompress)
	slog.Info("Added log file writer", "path", logFilePath, "max_size", logFileMaxSize, "max_backups", logFileMaxBackups, "compress", logFileCompress)

	return logFile
}

var initialized bool
var subs config.ConfigSubscriber

func SetLogLevel(level slog.Level) {
	logLevel.Set(level)
}

func updateLogger(cfg *config.Config) io.Writer {
	loggerMu.Lock()
	defer loggerMu.Unlock()

	slog.Info("Updating log writers...")
	oldFileLog := fileLog

	logToStdOut := cfg.Logging.ToStdout.Read()

	var writers []io.Writer = []io.Writer{}
	if logToStdOut {
		writers = append(writers, os.Stdout)
		slog.Info("Added Stdout writer")
	}

	fileLog = newLogFileWriter(cfg)
	if fileLog != nil {
		writers = append(writers, fileLog)
	}

	mw := io.MultiWriter(writers...)
	handler := slog.NewTextHandler(mw, &slog.HandlerOptions{
		Level: &logLevel,
	})
	slog.SetDefault(slog.New(handler))
	slog.Info("Log writers updated successfully")
	if oldFileLog != nil && oldFileLog != fileLog {
		if err := oldFileLog.Close(); err != nil {
			slog.Warn("Failed to close previous log file writer", "path", oldFileLog.Path(), "error", err)
		}
	}
	return mw
}

func Init(cfg *config.Config) {
	if initialized {
		return
	}
	initialized = true

	logLevel.Set(cfg.Logging.Level.Read())

	// Subscribe to log level changes
	subs.Add(cfg.Logging.Level.OnChange(func(newLevel slog.Level) {
		logLevel.Set(newLevel)
		slog.Info("Log level changed by configuration", "new_level", newLevel)
	}))

	subs.Add(cfg.Logging.File.OnChange(func(string) { updateLogger(cfg) }))
	subs.Add(cfg.Logging.MaxSize.OnChange(func(bytesize.ByteSize) { updateLogger(cfg) }))
	subs.Add(cfg.Logging.MaxBackups.OnChange(func(int) { updateLogger(cfg) }))
	subs.Add(cfg.Logging.Compress.OnChange(func(bool) { updateLogger(cfg) }))
	subs.Add(cfg.Logging.ToStdout.OnChange(func(bool) { updateLogger(cfg) }))

	slog.Info("Initializing logging...")

	slog.Info("Setting up log writers...")
	mw := updateLogger(cfg)

	// Write any early buffered log entries.
	early.EarlyBuffer.WriteTo(mw)
	early.EarlyBuffer = &bytes.Buffer{} // Reset buffer to "free" memory
}
