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
	config := config.Get()

	logFilePath := config.LogFile.Read()
	if logFilePath == "" {
		return nil, ErrNoLogFile
	}

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
func appendLogFileWriter(cfg *config.Config, writers *[]io.Writer) io.Writer {
	slog.Info("Initializing log file writer...")

	logFilePath := cfg.LogFile.Read()

	if logFilePath == "" {
		slog.Info("Log file logging is disabled, skipping log file writer initialization")
		return nil
	}

	tj := &timberjack.Logger{
		Filename:   logFilePath,
		MaxSize:    500,
		MaxBackups: 3,
		Compress:   true,
		LocalTime:  true,
	}

	*writers = append(*writers, tj)
	slog.Info("Added log file writer", "path", logFilePath)

	return tj
}

func SetLogLevel(level slog.Level) {
	logLevel.Set(level)
}

func Init() {
	config := config.Get()

	slog.Info("Initializing logging...")

	slog.Info("Setting up log writers...")
	var writers []io.Writer = []io.Writer{}
	if config.LogToStdio.Read() {
		writers = append(writers, os.Stdout)
		slog.Info("Added Stdout writer")
	}

	logFile := appendLogFileWriter(&config, &writers)

	level := config.LogLevel.Read()
	slog.Info("Setting log level", "log-level", level)
	logLevel.Set(level)

	slog.Info("Setting up slog handler...")
	mw := io.MultiWriter(writers...)
	handler := slog.NewTextHandler(mw, &slog.HandlerOptions{
		Level: &logLevel,
	})
	slog.SetDefault(slog.New(handler))

	// Write any buffered log entries to the log file
	early.EarlyBuffer.WriteTo(logFile)
	early.EarlyBuffer = &bytes.Buffer{} // Reset buffer to "free" memory

	slog.Info("Slog handler and multi-writer set up successfully", "log-level", level)
}
