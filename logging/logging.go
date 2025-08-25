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

	"github.com/DeRuina/timberjack"
)

var (
	ErrNoLogFile = errors.New("no log file configured")
)

var logLevel slog.LevelVar

func OpenLogFileRead() (*os.File, error) {
	cfgLock := config.Global.Immutable()

	var logFilePath string
	cfgLock.Read(func(c *config.Config) {
		logFilePath = c.LogFile.Read()
	})

	assertedPath, err := assertedpath.TryAssert(logFilePath)
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
func appendLogFileWriter(writers *[]io.Writer) io.Writer {
	slog.Info("Initializing log file writer...")

	cfgLock := config.Global.Immutable()

	var logFilePath string
	var logFileMaxSize int
	var logFileMaxBackups int
	var logFileCompress bool
	cfgLock.Read(func(c *config.Config) {
		logFilePath = c.LogFile.Read()
		logFileMaxSize = int(c.LogFileMaxSize.Read().MegaBytes())
		logFileMaxBackups = c.LogFileMaxBackups.Read()
		logFileCompress = c.LogFileCompress.Read()
	})

	if logFilePath == "" {
		slog.Info("Log file logging is disabled, skipping log file writer initialization")
		return nil
	}

	tj := &timberjack.Logger{
		Filename:   logFilePath,
		MaxSize:    logFileMaxSize,
		MaxBackups: logFileMaxBackups,
		Compress:   logFileCompress,
		LocalTime:  true,
	}

	*writers = append(*writers, tj)
	slog.Info("Added log file writer", "path", logFilePath, "max_size", logFileMaxSize, "max_backups", logFileMaxBackups, "compress", logFileCompress)

	return tj
}

func SetLogLevel(level slog.Level) {
	logLevel.Set(level)
}

func Init() {
	cfgLock := config.Global.Immutable()

	var logToStdOut bool
	cfgLock.Read(func(c *config.Config) {
		logToStdOut = c.LogToStdout.Read()

		// Subscribe to log level changes
		c.LogLevel.OnChange(func(newLevel slog.Level) {
			logLevel.Set(newLevel)
			slog.Info("Log level changed by configuration", "new_level", newLevel)
		})
		logLevel.Set(c.LogLevel.Read())
	})

	slog.Info("Initializing logging...")

	slog.Info("Setting up log writers...")
	var writers []io.Writer = []io.Writer{}
	if logToStdOut {
		writers = append(writers, os.Stdout)
		slog.Info("Added Stdout writer")
	}

	appendLogFileWriter(&writers)

	slog.Info("Setting up slog handler...")
	mw := io.MultiWriter(writers...)
	handler := slog.NewTextHandler(mw, &slog.HandlerOptions{
		Level: &logLevel,
	})
	slog.SetDefault(slog.New(handler))

	// Write any early buffered log entries.
	early.EarlyBuffer.WriteTo(mw)
	early.EarlyBuffer = &bytes.Buffer{} // Reset buffer to "free" memory

	slog.Info("Slog handler and multi-writer set up successfully", "log-level", logLevel.String())
}
