package logging

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"reservoir/config"
	"reservoir/logging/early"
	"reservoir/utils/assertedpath"
	"reservoir/utils/bytesize"
)

var (
	ErrNoLogFile = errors.New("no log file configured")
)

var fileLog *fileLogger = nil // Current file logger instance if any
var logLevel slog.LevelVar

func OpenLogFileRead() (*os.File, error) {
	assertedPath, err := assertedpath.TryAssert(fileLog.Path())
	if err != nil {
		return nil, err
	}

	logFile, err := os.OpenFile(assertedPath.Path, os.O_RDONLY, 0444)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	return logFile, nil
}

// Initializes and appends a log file writer to the provided writers slice if configured and returns it.
// If file logging is disabled in config, this will return nil.
func appendLogFileWriter(cfg *config.Config, writers *[]io.Writer) io.Writer {
	slog.Info("Initializing log file writer...")

	logFilePath := cfg.Logging.File.Read()
	logFileMaxSize := int(cfg.Logging.MaxSize.Read().MegaBytes())
	logFileMaxBackups := cfg.Logging.MaxBackups.Read()
	logFileCompress := cfg.Logging.Compress.Read()

	if logFilePath == "" {
		slog.Info("Log file logging is disabled, skipping log file writer initialization")
		return nil
	}

	fileLog = newFileLogger(logFilePath, logFileMaxSize, logFileMaxBackups, logFileCompress)

	*writers = append(*writers, fileLog)
	slog.Info("Added log file writer", "path", logFilePath, "max_size", logFileMaxSize, "max_backups", logFileMaxBackups, "compress", logFileCompress)

	return fileLog
}

var initialized bool
var subs config.ConfigSubscriber

func SetLogLevel(level slog.Level) {
	logLevel.Set(level)
}

func updateLogger(cfg *config.Config) io.Writer {
	slog.Info("Updating log writers...")

	logToStdOut := cfg.Logging.ToStdout.Read()

	var writers []io.Writer = []io.Writer{}
	if logToStdOut {
		writers = append(writers, os.Stdout)
		slog.Info("Added Stdout writer")
	}

	appendLogFileWriter(cfg, &writers)

	mw := io.MultiWriter(writers...)
	handler := slog.NewTextHandler(mw, &slog.HandlerOptions{
		Level: &logLevel,
	})
	slog.SetDefault(slog.New(handler))
	slog.Info("Log writers updated successfully")
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
